package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"
)

var config Config

type Config struct {
	FileName  string
	WebPort   string
	DNSname   string
	TestCheck string
	Duration  int
}

func checkDNS() []string {
	dt := time.Now()
	DNS := make([]string, 0)
	proto := []string{"tcp", "udp"}
	var checkDNS string
	for _, protocol := range proto {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 30 * time.Second,
				}
				return d.DialContext(ctx, protocol, config.DNSname+":53")
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Duration)*time.Millisecond)
		_, err := r.LookupHost(ctx, config.TestCheck)
		defer cancel()
		if err != nil {
			fmt.Println("["+dt.Format(time.RFC1123)+"]", err.Error())
			if strings.Contains(err.Error(), "timeout") {
				fmt.Println("["+dt.Format(time.RFC1123)+"]", protocol+" port 53 to DNS server "+config.DNSname+" closed!")
				checkDNS = "Bad " + config.DNSname + " " + "53/" + protocol
				DNS = append(DNS, checkDNS)
			}
		} else {
			fmt.Println("["+dt.Format(time.RFC1123)+"]", protocol+" port 53 to DNS server "+config.DNSname+" open!")
			checkDNS = "Good " + config.DNSname + " " + "53/" + protocol
			DNS = append(DNS, checkDNS)
		}
	}
	return DNS
}

func checkDNSudp() bool {
	dt := time.Now()
	var statDNS bool
	for _, ip := range checkDNS() {
		if strings.Contains(ip, "udp") {
			substr := ip
			sliceData := strings.Split(string(substr), " ")
			UDPstat := sliceData[0]
			if UDPstat == "Good" {
				fmt.Println("["+dt.Format(time.RFC1123)+"]", "Access to DNS-server "+config.DNSname+" ......... OK!")
				statDNS = true
			} else {
				fmt.Println("["+dt.Format(time.RFC1123)+"]", "Access to DNS-server "+config.DNSname+" ......... FAILED!")
				statDNS = false
			}
		}
	}
	return statDNS
}

func lookup() []string {
	dt := time.Now()
	sliceFullInfo := make([]string, 0)
	var resPORT, resNAME, checkIp string
	//	var wg sync.WaitGroup
	cha := make(chan string, 1000)

	fileBytes, err := ioutil.ReadFile(config.FileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sliceData := strings.Split(string(fileBytes), "\n")

	if checkDNSudp() == false {
	} else {
		for i := 0; i < len(sliceData); i++ {
			go func() {
				substr := sliceData[i]
				Data := strings.Split(string(substr), ":")
				resPORT = Data[1]
				resNAME = Data[0]
				r := &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						d := net.Dialer{
							Timeout: 10 * time.Second,
						}
						return d.DialContext(ctx, "udp", config.DNSname+":53")
					},
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Duration)*time.Millisecond)
				ip, err := r.LookupHost(ctx, resNAME)
				defer cancel()
				if err != nil {
					if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "timeout") {
						 fmt.Println("["+dt.Format(time.RFC1123)+"]", "Service name "+resNAME+" not resolved to IP.....")
						 fmt.Println(err.Error())
						checkIp = "Bad " + "no_resolve" + " " + resNAME + " " + resPORT
					}
				} else {
					for i := 0; i < len(ip); i++ {
						if i == 0 {
							fmt.Println("["+dt.Format(time.RFC1123)+"]", resNAME, "resolve to IP....:")
						}
						fmt.Println("Look", resNAME, ip[i])
						addr := ip[i]

						_, err := net.DialTimeout("tcp", addr+":"+resPORT, time.Duration(config.Duration)*time.Millisecond)
						if err == nil {
							checkIp = "Good " + addr + " " + resNAME + " " + resPORT
							checkIp += " " + checkIp + " "

						} else {
							checkIp = "Bad " + addr + " " + resNAME + " " + resPORT
							checkIp += " " + checkIp + " "

						}
					}

				}
				cha <- checkIp
			}()
			checkIp = <-cha
			sliceFullInfo = append(sliceFullInfo, checkIp)
		}
	}
	return sliceFullInfo
}

func init() {
	flag.IntVar(&config.Duration, "duration-lookup-a", 100, "Duration of the lookup resolve dns name query. The default is 100 Millisecond")
	flag.StringVar(&config.DNSname, "dns-server", "8.8.8.8", "You need to specify the IP address of the DNS server. The default is DNS 8.8.8.8")
	flag.StringVar(&config.TestCheck, "test-check", "www.google.ru", "Specify the address for testing the DNS server operation.. The default name www.google.ru")
	flag.StringVar(&config.FileName, "file-name", "domain_list.txt", "File Name for domain list. Should be in the directory of the executable. Default name domain_list.txt")
	flag.StringVar(&config.WebPort, "web-port", "9199", "Web port for show metrics")
	flag.Parse()
}

func main() {
	fmt.Println(lookup())
}
