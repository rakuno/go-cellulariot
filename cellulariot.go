package go_cellulariot

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	"github.com/tarm/serial"
)

// GPIO number
const (
	Bg96Enable   = 17
	UserButton   = 22
	Status       = 23
	Bg96Powerkey = 24
	UserLed      = 27
)

// ATCommand Parameter
const (
	TimeOut = 3
)

type cellulariotPins struct {
	pinBg96enable   rpio.Pin
	pinUserButton   rpio.Pin
	pinStatus       rpio.Pin
	pinBg96Powerkey rpio.Pin
	pinUserLed      rpio.Pin
}

type cellulariot struct {
	board      string
	ipAddress  string
	domainName string
	portNumber string
	timeout    time.Duration

	response string // variable for modem self.responses
	compose  string // variable for command self.composes

	cellulariotPins
	port *serial.Port
}

func (c *cellulariot) setupPins() {
	c.pinBg96enable = rpio.Pin(Bg96Enable)
	c.pinUserButton = rpio.Pin(UserButton)
	c.pinStatus = rpio.Pin(Status)
	c.pinBg96Powerkey = rpio.Pin(Bg96Powerkey)
	c.pinUserLed = rpio.Pin(UserLed)

	// RPi->Sixfab
	c.pinBg96enable.Output()
	c.pinBg96Powerkey.Output()
	c.pinUserLed.Output()
	// Sixfab->RPi
	c.pinStatus.Input()
	c.pinUserButton.Input()
}

func (c *cellulariot) setupGpio() {
	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Setup GPIO Pins
	c.setupPins()
}

func (c *cellulariot) cleanupGpio() {
	err := rpio.Close()
	if err != nil {
		log.Print("can not close GPIO...")
	}
}

func (c *cellulariot) closePort() {
	err := c.port.Close()
	if err != nil {
		log.Print("can not close serial...")
	}
}

func NewCellulariot() *cellulariot {
	conf := &serial.Config{Name: "/dev/ttyS0", Baud: 115200}
	serialPort, err := serial.OpenPort(conf)
	if err != nil {
		log.Fatal(err)
	}

	c := cellulariot{
		port:    serialPort,
		board:   "Sixfab Raspberry Pi Cellular Iot Shield",
		timeout: TimeOut,
	}
	c.setupGpio()

	return &c
}

func (c *cellulariot) DeleteCellulariot() {
	c.cleanupGpio()
	c.closePort()
}

/* GPIO APIs */
func (c *cellulariot) Enable() {
	c.pinBg96enable.Low()
	log.Print("BG96 module enabled!")
}

func (c *cellulariot) Disable() {
	c.pinBg96enable.High()
	log.Print("BG96 module disabled!")
}

// Function for sending at comamand to module
func (c *cellulariot) sendATCommandOnce(command string) {
	c.compose = ""
	c.compose = command + "\r"
	_, err := c.port.Write([]byte(c.compose))
	if err != nil {
		log.Printf("AT command write error, %s", err)
	}
	log.Print(c.compose)
}

// Function for sending at command to BG96_AT.
func (c *cellulariot) sendATComm(command, desiredResponse string) {
	var p []byte

	c.sendATCommandOnce(command)
	timer := time.Now()
	for {
		if (time.Now()).Sub(timer) > c.timeout {
			c.sendATCommandOnce(command)
			timer = time.Now()
		}
		c.response = ""
		readed := 0
		readedInitialbyte := false
		for {
			n, err := c.port.Read(p[readed : readed+1])
			if err != nil {
				log.Printf("error: Read failed: %s", err)
				break
			}
			if n > 0 && !readedInitialbyte {
				readedInitialbyte = true
			}
			if n == 0 && readedInitialbyte {
				break
			}
			readed += n
			if readed >= len(p) {
				break
			}
		}
		c.response = string(p)
		if strings.Index(c.response, desiredResponse) != -1 {
			log.Print(c.response)
			break
		}
	}

}
