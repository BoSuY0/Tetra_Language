//go:build !linux

package netrt

import "time"

const soReusePort = 0x0f

func ListenTCP4(cfg TCPListenConfig) (TCPListener, error) {
	return TCPListener{}, ErrUnsupported
}

func Accept(listenerFD int, cfg AcceptConfig) (int, error) {
	return -1, ErrUnsupported
}

func IsNonblocking(fd int) (bool, error) {
	return false, ErrUnsupported
}

func SetNonblocking(fd int, enabled bool) error {
	return ErrUnsupported
}

func Read(fd int, buf []byte) (int, error) {
	return 0, ErrUnsupported
}

func Recv(fd int, buf []byte) (int, error) {
	return 0, ErrUnsupported
}

func Write(fd int, buf []byte) (int, error) {
	return 0, ErrUnsupported
}

func Send(fd int, buf []byte) (int, error) {
	return 0, ErrUnsupported
}

func Writev(fd int, chunks [][]byte) (int, error) {
	return 0, ErrUnsupported
}

func Sendfile(outFD int, inFD int, offset *int64, count int) (int, error) {
	return 0, ErrUnsupported
}

func Close(fd int) error {
	return ErrUnsupported
}

func NewPoller() (*Poller, error) {
	return nil, ErrUnsupported
}

func (p *Poller) Close() error {
	return ErrUnsupported
}

func (p *Poller) Add(fd int, interest Interest) error {
	return ErrUnsupported
}

func (p *Poller) AddRead(fd int) error {
	return ErrUnsupported
}

func (p *Poller) AddReadWrite(fd int) error {
	return ErrUnsupported
}

func (p *Poller) Mod(fd int, interest Interest) error {
	return ErrUnsupported
}

func (p *Poller) Remove(fd int) error {
	return ErrUnsupported
}

func (p *Poller) Wait(maxEvents int, timeout time.Duration) ([]Event, error) {
	return nil, ErrUnsupported
}
