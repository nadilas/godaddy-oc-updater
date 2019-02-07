package kubeless

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/glendc/go-external-ip"
	"github.com/kubeless/kubeless/pkg/functions"
	"github.com/nadilas/godaddy-oc-updater/godaddy"
	"github.com/sirupsen/logrus"
)

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

func Handler(event functions.Event, ctx functions.Context) (string, error) {
	setup()
	domain := os.Getenv("API_DOMAIN")
	if domain == "" {
		panic("API_DOMAIN not set")
	}
	outbound, err := GetOutboundIP()
	if err != nil {
		panic(err)
	}
	logrus.Infof("Current IP: %s", outbound.String())
	currIP := outbound.String()

	dnsRecords, _, e := apiClient.V1Api.RecordGet(context.Background(), domain, "A", "", nil)
	if e != nil {
		panic(e)
	}

	var updates []godaddy.DnsRecordCreateType
	// check records
	for _, r := range dnsRecords {
		if r.Data != currIP {
			logrus.Warnf("Current IP has changed for %s. Updating %s -> %s", r.Name, r.Data, currIP)
			// simply replace it
			r.Data = currIP
			updates = append(updates, godaddy.DnsRecordCreateType{
				Data:     currIP,
				Name:     r.Name,
				Port:     r.Port,
				Priority: r.Priority,
				Protocol: r.Protocol,
				Service:  r.Service,
				Ttl:      r.Ttl,
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
			return msg, nil
		} else {
			logrus.Errorf("Failed to update dns records. %+v", replace)
			return "", errors.New("failed to update dns records")
		}
	} else {
		logrus.Infof("No update is required")
		spew.Dump(dnsRecords)
		return "no update is required", nil
	}
}
