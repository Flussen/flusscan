/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var ip string
var portStart int
var portEnd int
var timeoutSec int

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "scan ports by a ip",
	Long: `USE:
	flusscan scan -i (ip) -s (start_port) -e (end_port) -t (timeout)`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := scanPorts(ip, portStart, portEnd, time.Duration(timeoutSec)*time.Second); err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringVarP(&ip, "ip", "i", "", "ip addresses to scan")
	scanCmd.Flags().IntVarP(&portStart, "start", "s", 1, "initial port for the range to scan")
	scanCmd.Flags().IntVarP(&portEnd, "end", "e", 1024, "last port for the range to scan")
	scanCmd.Flags().IntVarP(&timeoutSec, "timeout", "t", 3, "timeout in seconds per connection attempt")
}

func scanPorts(ip string, start, end int, timeout time.Duration) error {

	if ip == "" {
		return errors.New("the ip cannot be empty, -i or --ip")
	}

	concurrencyLimit := 2000
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var ports []int

	for port := start; port <= end; port++ {
		sem <- struct{}{}
		wg.Add(1)
		go func(p int) {
			defer wg.Done()

			address := fmt.Sprintf("%s:%d", ip, p)
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err != nil {
				fmt.Printf("Port %d closed or filtered\n", p)
			} else {
				fmt.Printf("Port %d is OPEN!\n", p)
				mutex.Lock()
				ports = append(ports, p)
				mutex.Unlock()
				conn.Close()
			}
			<-sem
		}(port)
	}
	wg.Wait()

	fmt.Printf("OPEN PORTS: %v\n", ports)
	fmt.Printf("banner grapping.")
	time.Sleep(time.Millisecond * 300)
	fmt.Print(".")
	time.Sleep(time.Millisecond * 300)
	fmt.Print(".")

	for _, p := range ports {
		address := fmt.Sprintf("%s:%d", ip, p)
		conn, err := net.DialTimeout("tcp", address, time.Second*10)
		if err == nil {
			defer conn.Close()
			fmt.Printf("connected to port %d\n", p)

			if p == 80 || p == 443 {
				fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", ip)
			}

			buffer := make([]byte, 4096)
			conn.SetReadDeadline(time.Now().Add(timeout))
			n, err := conn.Read(buffer)

			if err == nil && n > 0 {
				fmt.Printf("Port %d banner: %s\n", p, buffer[:n])
			} else if err != nil {
				fmt.Printf("Error reading from port %d: %s\n", p, err)
			}
		} else {
			fmt.Printf("Error connecting to port %d: %s\n", p, err)
		}
	}
	return nil
}
