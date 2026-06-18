package specs

func tetraSource(category string) string {
	switch category {
	case "integer loops":
		return `module p25.integer_loops

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + (i % 7)
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	case "slice sum":
		return `module p25.slice_sum

func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    var r: Int = 0
    while r < 64:
        i = 0
        while i < n:
            total = total + xs[i]
            i = i + 1
        r = r + 1
    if total > 0:
        return 0
    return 1
`
	case "bounds-check loops":
		return `module p25.bounds_check_loops

func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 200000:
        let idx: Int = (i * 17) % n
        total = total + xs[idx]
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	case "function calls":
		return `module p25.function_calls

func mix(a: Int, b: Int) -> Int:
    return (a * 3 + b) % 97

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + mix(i, total)
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	case "recursion":
		return `module p25.recursion

func fib(n: Int) -> Int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 40:
        total = total + fib(10)
        i = i + 1
    if total == 2200:
        return 0
    return 1
`
	case "matrix multiply":
		return `module p25.matrix_multiply

func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var b: []i32 = core.make_i32(9)
    var c: []i32 = core.make_i32(9)
    var i: Int = 0
    while i < 9:
        a[i] = i + 1
        b[i] = 9 - i
        c[i] = 0
        i = i + 1
    var checksum: Int = 0
    var r: Int = 0
    while r < 2000:
        var row: Int = 0
        while row < 3:
            var col: Int = 0
            while col < 3:
                var k: Int = 0
                var total: Int = 0
                while k < 3:
                    total = total + a[row * 3 + k] * b[k * 3 + col]
                    k = k + 1
                c[row * 3 + col] = total
                col = col + 1
            row = row + 1
        checksum = checksum + c[r % 9]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`
	case "hash table":
		return `module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        keys[i] = i * 2 + 1
        values[i] = i + 7
        i = i + 1
    var checksum: Int = 0
    var q: Int = 0
    while q < n:
        let key: Int = q * 2 + 1
        checksum = checksum + lookup(keys, values, n, key)
        q = q + 1
    if checksum > 0:
        return 0
    return 1
`
	case "allocation":
		return `module p25.allocation

func main() -> Int
uses alloc, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 1024:
        var xs: []i32 = core.make_i32(32)
        xs[0] = r
        checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`
	case "region/island allocation":
		return `module p25.region_island_allocation

func main() -> Int
uses alloc, islands, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 256:
        island(256) as isl:
            var xs: []i32 = core.island_make_i32(isl, 16)
            xs[0] = r
            checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`
	case "JSON parse/stringify":
		return `module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    dst[0] = 123
    dst[1] = 34
    dst[2] = 109
    dst[3] = 101
    dst[4] = 115
    dst[5] = 115
    dst[6] = 97
    dst[7] = 103
    dst[8] = 101
    dst[9] = 34
    dst[10] = 58
    dst[11] = 34
    dst[12] = 72
    dst[13] = 101
    dst[14] = 108
    dst[15] = 108
    dst[16] = 111
    dst[17] = 44
    dst[18] = 32
    dst[19] = 87
    dst[20] = 111
    dst[21] = 114
    dst[22] = 108
    dst[23] = 100
    dst[24] = 33
    dst[25] = 34
    dst[26] = 125
    return 27

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        total = total + write_message_object(buf)
        i = i + 1
    if total == 55296:
        return 0
    return 1
`
	case "HTTP plaintext/json":
		return `module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 72
    dst[20] = 101
    dst[21] = 108
    dst[22] = 108
    dst[23] = 111
    return 24

func write_json_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 123
    dst[20] = 125
    return 21

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    var json_buf: []u8 = core.make_u8(192)
    var i: Int = 0
    var total: Int = 0
    while i < 1024:
        total = total + write_plaintext_response(plain)
        total = total + write_json_response(json_buf)
        i = i + 1
    if total > 0:
        return 0
    return 1
`
	case "PostgreSQL single/multiple/update":
		return `module p25.postgresql_single_multiple_update

func frame_data_row() -> Int:
    return 68

func frame_payload_start(offset: Int) -> Int:
    return offset + 5

func frame_type_at(src: []u8, offset: Int) -> Int
uses mem:
    return src[offset]

func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 16777216) % 256
    dst[start + 1] = (value / 65536) % 256
    dst[start + 2] = (value / 256) % 256
    dst[start + 3] = value % 256
    return start + 4

func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 256) % 256
    dst[start + 1] = value % 256
    return start + 2

func main() -> Int
uses alloc, mem:
    var frame: []u8 = core.make_u8(64)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        frame[0] = frame_data_row()
        var pos: Int = write_i32_be_at(frame, 1, 12)
        pos = write_i16_be_at(frame, pos, 2)
        total = total + frame_type_at(frame, 0) + frame_payload_start(0)
        i = i + 1
    if total > 0:
        return 0
    return 1
`
	case "actor ping-pong":
		return `func pong() -> i32
uses actors:
    var v: i32 = core.recv()
    if v == 41:
        var _sent: i32 = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> i32
uses actors:
    var p: actor = core.spawn("pong")
    var _sent: i32 = core.send(p, 41)
    var r: i32 = core.recv()
    if r == 42:
        return 0
    return 1
`
	case "parallel map/reduce":
		return `module p25.parallel_map_reduce

func left_worker() -> Int:
    return 13

func mid_worker() -> Int:
    return 17

func right_worker() -> Int:
    return 12

func main() -> Int
uses runtime:
    let left: task.i32 = core.task_spawn_i32("left_worker")
    let mid: task.i32 = core.task_spawn_i32("mid_worker")
    let right: task.i32 = core.task_spawn_i32("right_worker")
    let total: Int = core.task_join_i32(left) + core.task_join_i32(mid) + core.task_join_i32(right)
    if total == 42:
        return 0
    return total
`
	case "startup time", "binary size":
		return `module p25.` + slug(category) + `

func main() -> Int:
    return 0
`
	case "compile time":
		return `module p25.compile_time

func f0(x: Int) -> Int:
    return x + 1

func f1(x: Int) -> Int:
    return f0(x) * 3

func f2(x: Int) -> Int:
    return f1(x) + f0(x)

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + f2(i)
        i = i + 1
    if total == 0:
        return 1
    return 0
`
	default:
		return `module p25.default

func main() -> Int:
    return 0
`
	}
}
