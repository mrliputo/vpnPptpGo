package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	// Konfigurasi koneksi VPN PPTP
	vpnConfig := map[string]string{
		"server":   "?????",
		"username": "?????",
		"password": "????",
	}

	// Menjalankan koneksi VPN PPTP
	err := connectVPN(vpnConfig)
	if err != nil {
		log.Fatal("Gagal terhubung ke VPN:", err)
	}

	// Mendapatkan alamat IP ppp0
	ppp0IP, err := getPPPIpAddress()
	if err != nil {
		log.Fatal("Gagal mendapatkan alamat IP ppp0:", err)
	}
	fmt.Println("Alamat IP ppp0:", ppp0IP)

	client := &http.Client{}

	// Create a transport with the desired interface
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(ppp0IP)},
		}
		return dialer.DialContext(ctx, network, addr)
	}
	client.Transport = transport

	// Create a GET request
	req, err := http.NewRequest("GET", "https://10.245.192.37", nil)
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return
	}
	req.Header.Set("Interface", "ppp0")
	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response:", err)
		return
	}

	// Print the response body
	fmt.Println("Response:", string(body))
}

// Fungsi untuk menghubungkan ke VPN PPTP
func connectVPN(config map[string]string) error {
	// Menghentikan service NetworkManager (opsional, tergantung pada sistem operasi)
	err := exec.Command("systemctl", "stop", "NetworkManager").Run()
	if err != nil {
		return err
	}

	// Menjalankan perintah untuk menghubungkan ke VPN PPTP
	cmd := exec.Command("pptpsetup", "--create", "myvpn", "--server", config["server"], "--username", config["username"], "--password", config["password"], "--encrypt", "--start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// Fungsi untuk mendapatkan alamat IP dari antarmuka ppp0
func getPPPIpAddress() (string, error) {
	output, err := exec.Command("ip", "addr", "show", "ppp0").Output()
	if err != nil {
		return "", err
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				ip := strings.Split(fields[1], "/")
				return ip[0], nil
			}
		}
	}

	return "", fmt.Errorf("tidak dapat menemukan alamat IP ppp0")
}
