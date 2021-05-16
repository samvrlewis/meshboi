package meshboi

import (
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

const (
	IFF_TUN   = 0x1    /* Flag to open a TUN device (rather than TAP) */
	IFF_NO_PI = 0x1000 /* Do not provide packet information */
)

type ifReq struct {
	Name  [16]byte
	Flags uint16
}

type Tun struct {
	io.ReadWriteCloser
	Name string
}

type TunConn interface {
	io.ReadWriteCloser
}

//https://www.kernel.org/doc/Documentation/networking/tuntap.txt
func NewTun(name string) (*Tun, error) {
	tunFile, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)

	if err != nil {
		return nil, err
	}
	req := ifReq{}
	req.Flags = IFF_TUN | IFF_NO_PI
	copy(req.Name[:], name)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, tunFile.Fd(), syscall.TUNSETIFF, uintptr(unsafe.Pointer(&req)))

	if errno != 0 {
		return nil, os.NewSyscallError("ioctl", errno)
	}

	tun := Tun{
		Name:            name,
		ReadWriteCloser: tunFile,
	}

	return &tun, nil
}

// Makes a Tun with the desired config and immediately sets it up
func NewTunWithConfig(name string, ip string, mtu int) (*Tun, error) {
	tun, err := NewTun(name)

	if err != nil {
		return nil, err
	}

	if err := tun.SetNetwork(ip); err != nil {
		return nil, err
	}

	if err := tun.SetMtu(mtu); err != nil {
		return nil, err
	}

	if err := tun.SetLinkUp(); err != nil {
		return nil, err
	}

	return tun, nil
}

func (t Tun) SetLinkUp() error {
	cmd := exec.Command("/sbin/ip", "link", "set", t.Name, "up")

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (t Tun) SetNetwork(ip string) error {
	cmd := exec.Command("/sbin/ip", "addr", "add", ip, "dev", t.Name)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (t Tun) SetMtu(mtu int) error {
	cmd := exec.Command("/sbin/ip", "link", "set", "dev", t.Name, "mtu", strconv.Itoa(mtu))

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
