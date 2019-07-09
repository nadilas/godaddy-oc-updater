package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/glendc/go-external-ip"
	"github.com/kryptoslogic/godaddy-domainclient"
	"github.com/sirupsen/logrus"
)

var Version string

func main() {
	handler()
}

var (
	apiClient *godaddy.APIClient
)

func setup() {
	basePath := os.Getenv("API_BASE")
	if basePath == "" {
		panic("API_BASE not set")
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		panic("API_KEY not set")
	}
	apiSecret := os.Getenv("API_SECRET")
	if apiSecret == "" {
		panic("API_SECRET not set")
	}
	var apiConfig = godaddy.NewConfiguration()
	// Test
	// apiConfig.BasePath = "https://api.ote-godaddy.com/"

	// Prod
	// apiConfig.BasePath = "https://api.godaddy.com/"

	// set from env
	apiConfig.BasePath = basePath

	// Set auth
	authString := fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret)

	// Set auth production
	apiConfig.AddDefaultHeader("Authorization", authString)

	apiClient = godaddy.NewAPIClient(apiConfig)
}

// Get preferred outbound ip of this machine
func GetOutboundIP() (net.IP, error) {
	// Create the default consensus,
	// using the default configuration and no logger.
	consensus := externalip.DefaultConsensus(nil, nil)
	// Get your IP,
	// which is never <nil> when err is <nil>.
	return consensus.ExternalIP()
}

func handler() (string, error) {
	setup()
	var whitelist []string
	allowedNamesStr := os.Getenv("DOMAIN_NAMES_WHITELIST")
	if allowedNamesStr == "" {
		logrus.Infof("No whitelist provided, update will consider all A records for update.")
	} else {
		whitelist = strings.Split(allowedNamesStr, ",")
	}
	domain := os.Getenv("API_DOMAIN")
	if domain == "" {
		panic("API_DOMAIN not set")
	}
	ttlStr := os.Getenv("API_NEW_TTL")
	var (
		ttl int
		e   error
	)
	if ttlStr != "" {
		ttl, e = strconv.Atoi(ttlStr)
		if e != nil {
			logrus.Errorf("Cannot interpret TTL input: %s", e)
			ttl = 3600
		} else {
			logrus.Infof("Setting TTL to: %d", ttl)
		}
	}
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	ipv4 := string(bodyBytes)
	ipv6IP, _ := GetOutboundIP()
	ipv6 := ""
	if !strings.Contains(ipv6IP.String(), ".") {
		ipv6 = ipv6IP.String() // no ipv6 is available
	}
	// io.Copy(os.Stdout, resp.Body)
	// returns only ipv6 ?
	// outbound, err := GetOutboundIP()
	// if err != nil {
	//	panic(err)
	//}
	logrus.Infof("Current ipv4 IP: %s", ipv4)
	currIP := ipv4

	dnsRecords, _, e := apiClient.V1Api.RecordGet(context.Background(), domain, "A", "", nil)
	if e != nil {
		panic(e)
	}

	var updates []godaddy.DnsRecordCreateType
	// check records
	for _, r := range dnsRecords {
		if len(whitelist) > 0 && !strArrContains(whitelist, r.Name) {
			logrus.Warnf("%s is not in allowed names list, hence will not be updated.", r.Name)
			// put back as is
			updates = append(updates, godaddy.DnsRecordCreateType{
				Data:     r.Data,
				Name:     r.Name,
				Port:     r.Port,
				Priority: r.Priority,
				Protocol: r.Protocol,
				Service:  r.Service,
				Ttl:      r.Ttl,
				Weight:   r.Weight,
			})
			continue
		}
		if r.Data != currIP || r.Ttl != int32(ttl) {
			logrus.Warnf("Current IP has changed for %s. Updating %s -> %s", r.Name, r.Data, currIP)
			// simply replace it
			r.Data = currIP
			if r.Ttl != int32(ttl) {
				logrus.Infof("Updating TTL %d -> %d", r.Ttl, ttl)
			}
			updates = append(updates, godaddy.DnsRecordCreateType{
				Data:     currIP,
				Name:     r.Name,
				Port:     r.Port,
				Priority: r.Priority,
				Protocol: r.Protocol,
				Service:  r.Service,
				Ttl:      int32(ttl),
				Weight:   r.Weight,
			})
		}
	}
	if len(updates) > 0 {
		replace, e := apiClient.V1Api.RecordReplaceType(context.Background(), domain, "A", updates, nil)
		if e != nil {
			logrus.Errorf("Failed to replace old records. %s", e.(godaddy.GenericSwaggerError).Body())
			panic(e)
		}

		if replace.StatusCode == 200 {
			msg := fmt.Sprintf("Successfully updated %d dns records.", len(updates))
			logrus.Infof(msg)
		} else {
			logrus.Errorf("Failed to update dns records. %+v", replace)
			return "", errors.New("failed to update dns records")
		}
	} else {
		logrus.Infof("No ipv4 update is required")
		//spew.Dump(dnsRecords)
	}

	if ipv6 == "" {
		return "no ipv6 update is need", nil // done
	}
	logrus.Infof("Current ipv6 IP: %s", ipv6)
	currIP = ipv6

	dnsRecords, _, e = apiClient.V1Api.RecordGet(context.Background(), domain, "AAAA", "", nil)
	if e != nil {
		panic(e)
	}

	var updates2 []godaddy.DnsRecordCreateType
	// check records
	for _, r := range dnsRecords {
		if len(whitelist) > 0 && !strArrContains(whitelist, r.Name) {
			logrus.Warnf("%s is not in allowed names list, hence will not be updated.", r.Name)
			// put back as is
			updates2 = append(updates2, godaddy.DnsRecordCreateType{
				Data:     r.Data,
				Name:     r.Name,
				Port:     r.Port,
				Priority: r.Priority,
				Protocol: r.Protocol,
				Service:  r.Service,
				Ttl:      r.Ttl,
				Weight:   r.Weight,
			})
			continue
		}

		if r.Data != currIP || r.Ttl != int32(ttl) {
			logrus.Warnf("Current IP has changed for %s. Updating %s -> %s", r.Name, r.Data, currIP)
			// simply replace it
			r.Data = currIP
			if r.Ttl != int32(ttl) {
				logrus.Infof("Updating TTL %d -> %d", r.Ttl, ttl)
			}
			updates2 = append(updates2, godaddy.DnsRecordCreateType{
				Data:     currIP,
				Name:     r.Name,
				Port:     r.Port,
				Priority: r.Priority,
				Protocol: r.Protocol,
				Service:  r.Service,
				Ttl:      int32(ttl),
				Weight:   r.Weight,
			})
		}
	}
	if len(updates2) > 0 {
		replace, e := apiClient.V1Api.RecordReplaceType(context.Background(), domain, "AAAA", updates2, nil)
		if e != nil {
			logrus.Errorf("Failed to replace old records. %s", e.(godaddy.GenericSwaggerError).Body())
			panic(e)
		}

		if replace.StatusCode == 200 {
			msg := fmt.Sprintf("Successfully updated %d dns records.", len(updates2))
			logrus.Infof(msg)
			return msg, nil
		} else {
			logrus.Errorf("Failed to update dns records. %+v", replace)
			return "", errors.New("failed to update ipv6 dns records")
		}
	} else {
		logrus.Infof("No ipv6 update is required")
		//spew.Dump(dnsRecords)
		return "no update is required", nil
	}
}

// Determines if a string is part of the array
func strArrContains(arr []string, s string) bool {
	for _, str := range arr {
		if str == s {
			return true
		}
	}
	return false
}
