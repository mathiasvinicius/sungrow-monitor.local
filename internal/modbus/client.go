package modbus

import (
	"fmt"
	"sync"
	"time"

	"github.com/simonvetter/modbus"
)

type Client struct {
	client  *modbus.ModbusClient
	mu      sync.Mutex
	ip      string
	port    int
	slaveID uint8
	timeout time.Duration
}

func NewClient(ip string, port int, slaveID uint8, timeout time.Duration) *Client {
	return &Client{
		ip:      ip,
		port:    port,
		slaveID: slaveID,
		timeout: timeout,
	}
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		return nil
	}

	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:     fmt.Sprintf("tcp://%s:%d", c.ip, c.port),
		Timeout: c.timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create modbus client: %w", err)
	}

	if err := client.Open(); err != nil {
		return fmt.Errorf("failed to connect to inverter: %w", err)
	}

	client.SetUnitId(c.slaveID)
	c.client = client

	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil
	}

	err := c.client.Close()
	c.client = nil
	return err
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client != nil
}

func (c *Client) ReadInputRegisters(address uint16, quantity uint16) ([]uint16, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	regs, err := c.client.ReadRegisters(address, quantity, modbus.INPUT_REGISTER)
	if err != nil {
		return nil, fmt.Errorf("failed to read input registers at %d: %w", address, err)
	}

	return regs, nil
}

func (c *Client) ReadHoldingRegisters(address uint16, quantity uint16) ([]uint16, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	regs, err := c.client.ReadRegisters(address, quantity, modbus.HOLDING_REGISTER)
	if err != nil {
		return nil, fmt.Errorf("failed to read holding registers at %d: %w", address, err)
	}

	return regs, nil
}

func (c *Client) ReadUint16(address uint16) (uint16, error) {
	regs, err := c.ReadInputRegisters(address, 1)
	if err != nil {
		return 0, err
	}
	return regs[0], nil
}

func (c *Client) ReadInt16(address uint16) (int16, error) {
	regs, err := c.ReadInputRegisters(address, 1)
	if err != nil {
		return 0, err
	}
	return int16(regs[0]), nil
}

func (c *Client) ReadUint32(address uint16) (uint32, error) {
	regs, err := c.ReadInputRegisters(address, 2)
	if err != nil {
		return 0, err
	}
	// Little-endian: low word first, high word second
	return uint32(regs[0]) | uint32(regs[1])<<16, nil
}

func (c *Client) ReadInt32(address uint16) (int32, error) {
	val, err := c.ReadUint32(address)
	if err != nil {
		return 0, err
	}
	return int32(val), nil
}

func (c *Client) ReadString(address uint16, length uint16) (string, error) {
	regs, err := c.ReadInputRegisters(address, length)
	if err != nil {
		return "", err
	}

	bytes := make([]byte, 0, length*2)
	for _, reg := range regs {
		bytes = append(bytes, byte(reg>>8), byte(reg&0xFF))
	}

	// Remove null bytes
	for len(bytes) > 0 && bytes[len(bytes)-1] == 0 {
		bytes = bytes[:len(bytes)-1]
	}

	return string(bytes), nil
}

func (c *Client) Reconnect() error {
	c.Close()
	return c.Connect()
}
