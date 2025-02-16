package main

import (
	"errors"
	"net"
	"sort"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"
)

const Hostname = "wopr.norad.org"
const Prefix = "/chatgpt/dns-actions"

func main() {
	s := fuego.NewServer(
		fuego.WithAddr("0.0.0.0:8080"),
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(
				fuego.OpenAPIConfig{

				}
			),
		),
		//fuego.WithBasePath("/chatgpt/actions/dns"),
		// fuego.WithEngineOptions(
		// 	fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
		// 		JSONFilePath: "doc/openapi.json",
		// 		SpecURL:      "https://wopr.norad.org/chatgpt/actions/dns/swagger/openapi.json",
		// 		SwaggerURL:   "https://wopr.norad.org/chatgpt/actions/dns/swagger",
		// 		UIHandler:    fuego.DefaultOpenAPIHandler,
		// 	}),
		// ),
	)

	fuego.Get(s, "/mx", getMX,
		option.OperationID("getMX"),
		option.Summary("List the MX records for a single domain"),
		option.Description("List the MX records for a single domain. For many domains, use the bulk endpoint instead."),
		option.Query("domain", "The domain name to look at", param.Required()),
	)

	fuego.Post(s, "/mx/bulk", getMXBulk,
		option.OperationID("getMXBulk"),
		option.Summary("List the MX records for many domains"),
		option.Description("List the MX records for up to 50 domains."),
	)

	s.Run()
}

type MXHost struct {
	Host       string   `json:"host"`
	Preference uint16   `json:"preference"`
	Addresses  []string `json:"addresses"`
}

func fetchMXRecords(domain string) ([]MXHost, error) {
	records, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}

	result := []MXHost{}
	for _, record := range records {
		// Look up IP addresses for each MX host
		addresses, err := net.LookupHost(record.Host)
		if err != nil {
			// Skip this record if we can't resolve addresses
			continue
		}

		result = append(result, MXHost{
			Host:       record.Host,
			Preference: record.Pref,
			Addresses:  addresses,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Preference != result[j].Preference {
			return result[i].Preference < result[j].Preference
		}
		return result[i].Host < result[j].Host
	})

	return result, nil
}

func getMX(c fuego.ContextNoBody) ([]MXHost, error) {
	domain := c.QueryParam("domain")
	return fetchMXRecords(domain)
}

type GetMXBulkInput struct {
	Domains []string `json:"domains"`
}

func getMXBulk(input fuego.ContextWithBody[GetMXBulkInput]) (map[string][]MXHost, error) {
	body, err := input.Body()
	if err != nil {
		return nil, err
	}

	if len(body.Domains) == 0 {
		return nil, errors.New("no domains provided")
	}

	if len(body.Domains) > 50 {
		return nil, errors.New("too many domains: maximum 50 domains allowed")
	}

	result := map[string][]MXHost{}

	for _, domain := range body.Domains {
		records, err := fetchMXRecords(domain)
		if err == nil {
			result[domain] = records
		}
	}

	return result, nil
}
