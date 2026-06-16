package webrt

import (
	"context"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/htmlrt"
	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/jsonrt"
	"tetra_language/compiler/internal/pgrt"
	"tetra_language/internal/outputformat"
)

type messagePayload struct {
	Message string `json:"message"`
}

type worldPayload struct {
	ID           int `json:"id"`
	RandomNumber int `json:"randomNumber"`
}

func JSONMessageHandler(message string) httprt.Handler {
	return func(req httprt.Request) httprt.Response {
		return negotiatedJSONPayloadResponse(
			req,
			jsonrt.AppendMessageObject(nil, message),
			messagePayload{Message: message},
		)
	}
}

func DBHandler(pool *pgrt.Pool, nextID func() int) httprt.Handler {
	return func(req httprt.Request) httprt.Response {
		world, err := fetchWorld(context.Background(), pool, nextWorldID(nextID))
		if err != nil {
			return dbErrorResponse()
		}
		return negotiatedJSONPayloadResponse(
			req,
			jsonrt.AppendWorldObject(nil, world.ID, world.RandomNumber),
			worldPayload{ID: world.ID, RandomNumber: world.RandomNumber},
		)
	}
}

func QueriesHandler(pool *pgrt.Pool, nextID func() int) httprt.Handler {
	return func(req httprt.Request) httprt.Response {
		count := NormalizeQueryCount(req.QueryValue("queries"))
		worlds := make([]jsonrt.World, 0, count)
		conn, err := pool.Checkout(context.Background())
		if err != nil {
			return dbErrorResponse()
		}
		releaseErr := error(nil)
		defer func() {
			_ = conn.Release(releaseErr)
		}()
		for i := 0; i < count; i++ {
			world, err := queryWorld(context.Background(), conn.Conn, nextWorldID(nextID))
			if err != nil {
				releaseErr = err
				return dbErrorResponse()
			}
			worlds = append(worlds, world)
		}
		return negotiatedJSONPayloadResponse(req, jsonrt.AppendWorldArray(nil, worlds), worldPayloads(worlds))
	}
}

func UpdatesHandler(pool *pgrt.Pool, nextID func() int, nextRandom func() int) httprt.Handler {
	return func(req httprt.Request) httprt.Response {
		count := NormalizeQueryCount(req.QueryValue("queries"))
		worlds := make([]jsonrt.World, 0, count)
		conn, err := pool.Checkout(context.Background())
		if err != nil {
			return dbErrorResponse()
		}
		releaseErr := error(nil)
		defer func() {
			_ = conn.Release(releaseErr)
		}()
		usedRandoms := map[int]bool{}
		for i := 0; i < count; i++ {
			world, err := queryWorld(context.Background(), conn.Conn, nextWorldID(nextID))
			if err != nil {
				releaseErr = err
				return dbErrorResponse()
			}
			world.RandomNumber = nextUniqueRandomNumber(nextRandom, usedRandoms)
			if err := updateWorld(context.Background(), conn.Conn, world); err != nil {
				releaseErr = err
				return dbErrorResponse()
			}
			worlds = append(worlds, world)
		}
		return negotiatedJSONPayloadResponse(req, jsonrt.AppendWorldArray(nil, worlds), worldPayloads(worlds))
	}
}

func FortunesHandler(pool *pgrt.Pool) httprt.Handler {
	return func(req httprt.Request) httprt.Response {
		fortunes, err := fetchFortunes(context.Background(), pool)
		if err != nil {
			return dbErrorResponse()
		}
		fortunes = append(fortunes, htmlrt.Fortune{
			ID:      0,
			Message: "Additional fortune added at request time.",
		})
		return httprt.Response{
			StatusCode:  200,
			ContentType: "text/html; charset=utf-8",
			Body:        htmlrt.RenderFortunes(nil, fortunes),
		}
	}
}

func NormalizeQueryCount(raw string) int {
	count, err := strconv.Atoi(raw)
	if err != nil || count < 1 {
		return 1
	}
	if count > 500 {
		return 500
	}
	return count
}

func negotiatedJSONPayloadResponse(req httprt.Request, jsonBody []byte, toonValue any) httprt.Response {
	if acceptExplicitTOON(req.Header("Accept")) {
		body, err := outputformat.MarshalStructured(outputformat.TOON, toonValue)
		if err != nil {
			return dbErrorResponse()
		}
		return httprt.Response{
			StatusCode:  200,
			ContentType: outputformat.MediaTypeTOON,
			Body:        body,
		}
	}
	return httprt.Response{
		StatusCode:  200,
		ContentType: outputformat.MediaTypeJSON,
		Body:        jsonBody,
	}
}

func acceptExplicitTOON(raw string) bool {
	for _, part := range strings.Split(raw, ",") {
		media, q := parseAcceptMediaRange(part)
		if media == "text/toon" && q > 0 {
			return true
		}
	}
	return false
}

func parseAcceptMediaRange(raw string) (string, float64) {
	q := 1.0
	parts := strings.Split(raw, ";")
	media := strings.ToLower(strings.TrimSpace(parts[0]))
	for _, param := range parts[1:] {
		key, value, ok := strings.Cut(param, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || parsed < 0 {
			return media, 0
		}
		if parsed > 1 {
			parsed = 1
		}
		q = parsed
	}
	return media, q
}

func worldPayloads(worlds []jsonrt.World) []worldPayload {
	payloads := make([]worldPayload, 0, len(worlds))
	for _, world := range worlds {
		payloads = append(payloads, worldPayload{ID: world.ID, RandomNumber: world.RandomNumber})
	}
	return payloads
}

func fetchWorld(ctx context.Context, pool *pgrt.Pool, id int) (jsonrt.World, error) {
	conn, err := pool.Checkout(ctx)
	if err != nil {
		return jsonrt.World{}, err
	}
	releaseErr := error(nil)
	defer func() {
		_ = conn.Release(releaseErr)
	}()
	world, err := queryWorld(ctx, conn.Conn, id)
	if err != nil {
		releaseErr = err
		return jsonrt.World{}, err
	}
	return world, nil
}

func queryWorld(ctx context.Context, conn *pgrt.Conn, id int) (jsonrt.World, error) {
	result, err := conn.PreparedQueryFormat(
		ctx,
		"world_by_id",
		"SELECT id, randomNumber FROM World WHERE id=$1",
		[]uint32{pgrt.Int4OID},
		[]int16{pgrt.BinaryFormat},
		[][]byte{pgrt.AppendInt4Binary(nil, id)},
		nil,
	)
	if err != nil {
		return jsonrt.World{}, err
	}
	if len(result.Rows) == 0 {
		return jsonrt.World{}, pgrt.ErrUnexpectedMessage
	}
	row := result.Rows[0]
	worldID, err := decodeWorldInt4(result.Columns, row, 0)
	if err != nil {
		return jsonrt.World{}, err
	}
	randomNumber, err := decodeWorldInt4(result.Columns, row, 1)
	if err != nil {
		return jsonrt.World{}, err
	}
	return jsonrt.World{ID: worldID, RandomNumber: randomNumber}, nil
}

func updateWorld(ctx context.Context, conn *pgrt.Conn, world jsonrt.World) error {
	_, err := conn.PreparedQueryFormat(
		ctx,
		"update_world_random",
		"UPDATE World SET randomNumber=$1 WHERE id=$2",
		[]uint32{pgrt.Int4OID, pgrt.Int4OID},
		[]int16{pgrt.BinaryFormat},
		[][]byte{pgrt.AppendInt4Binary(nil, world.RandomNumber), pgrt.AppendInt4Binary(nil, world.ID)},
		nil,
	)
	return err
}

func decodeWorldInt4(columns []pgrt.Column, row pgrt.Row, index int) (int, error) {
	if index < 0 || index >= len(row) {
		return 0, pgrt.ErrUnexpectedMessage
	}
	format := pgrt.TextFormat
	if index < len(columns) {
		format = columns[index].Format
	}
	return pgrt.DecodeInt4(row[index], format)
}

func fetchFortunes(ctx context.Context, pool *pgrt.Pool) ([]htmlrt.Fortune, error) {
	conn, err := pool.Checkout(ctx)
	if err != nil {
		return nil, err
	}
	releaseErr := error(nil)
	defer func() {
		_ = conn.Release(releaseErr)
	}()
	result, err := conn.Conn.PreparedQuery(ctx, "all_fortunes", "SELECT id, message FROM Fortune", nil, nil)
	if err != nil {
		releaseErr = err
		return nil, err
	}
	fortunes := make([]htmlrt.Fortune, 0, len(result.Rows))
	for _, row := range result.Rows {
		id, err := strconv.Atoi(row.String(0))
		if err != nil {
			releaseErr = err
			return nil, err
		}
		fortunes = append(fortunes, htmlrt.Fortune{
			ID:      id,
			Message: row.String(1),
		})
	}
	return fortunes, nil
}

func nextWorldID(nextID func() int) int {
	if nextID == nil {
		return 1
	}
	id := nextID()
	if id < 1 {
		return 1
	}
	if id > 10000 {
		return 10000
	}
	return id
}

func nextUniqueRandomNumber(nextRandom func() int, used map[int]bool) int {
	for attempts := 0; attempts < 10000; attempts++ {
		value := nextWorldID(nextRandom)
		if !used[value] {
			used[value] = true
			return value
		}
	}
	for value := 1; value <= 10000; value++ {
		if !used[value] {
			used[value] = true
			return value
		}
	}
	return 1
}

func dbErrorResponse() httprt.Response {
	return httprt.Response{
		StatusCode: 500,
		Body:       []byte("Internal Server Error"),
		KeepAlive:  false,
	}
}
