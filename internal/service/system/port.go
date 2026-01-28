package system

import (
	"fmt"
	"net"
)

func CheckPorts(ports ...int) error {
	for _, port := range ports {
		if err := checkPort(port); err != nil {
			return err
		}
	}
	return nil
}

func checkPort(port int) error {
	addr := fmt.Sprintf(":%d", port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is not available: %w", port, err)
	}

	// 关键：一定要立刻关闭
	_ = l.Close()
	return nil
}
