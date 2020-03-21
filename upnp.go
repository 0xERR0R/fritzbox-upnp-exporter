package main

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

const IGDESCENDPOINT string = "/igddesc.xml"

type UPnPClient struct {
	URL string
}

func NewUPnPClient(url string) *UPnPClient {
	return &UPnPClient{
		URL: fmt.Sprintf("http://%s:49000", url),
	}
}

func (uc *UPnPClient) Execute() map[string]string {
	result := make(map[string]string)
	for _, service := range uc.parseServices() {
		for _, action := range uc.parseActions(service) {
			message := fmt.Sprintf(`
		<?xml version="1.0"?> 
        <s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" 
				s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"> 
            <s:Body><u:%s xmlns:u='%s'/></s:Body>
		</s:Envelope>`, action.Name, service.ServiceType)

			//		fmt.Printf("Calling %s with action %s\n", uc.URL+service.ControlURL, action.Name)
			//		fmt.Println(message)
			//dr := newRequest("POST", uc.URL+service.SCPDURL, message)
			dr := newRequest("POST", uc.URL+service.ControlURL, message)

			//dr := dac.NewRequest("admin", "!jryN18VD!", "POST", uc.URL+service.ControlURL, message)
			dr.Header.Add("Content-Type", "text/xml")
			dr.Header.Add("charset", "utf-8")
			dr.Header.Add("SoapAction", fmt.Sprintf("%s#%s", service.ServiceType, action.Name))
			//dr.Header.Add("Authorization", fmt.Sprintf("Basic %s", "YWRtaW46IWpyeU4xOFZEIQ=="))

			//dr1 := dac.NewRequest("admin", "!jryN18VD!", "POST", uc.URL+service.ControlURL, message)
			//	response1, err1 := dr1.Execute()

			// TODO close?
			decoder := xml.NewDecoder(do(dr))
			for {
				t, _ := decoder.Token()
				if t == nil {
					break
				}
				switch se := t.(type) {
				case xml.StartElement:
					for _, argument := range action.Arguments {
						if se.Name.Local == argument.Name {
							t, _ = decoder.Token()
							switch element := t.(type) {
							case xml.CharData:
								result[service.ServiceId+"_"+argument.RelatedStateVariable] = string(element)
							}
						}
					}
				}
			}
		}
	}
	printMap(result)
	return result
}

func printMap(m map[string]string) {
	var maxLenKey int
	for k, _ := range m {
		if len(k) > maxLenKey {
			maxLenKey = len(k)
		}
	}

	for k, v := range m {
		fmt.Println(k + ": " + strings.Repeat(" ", maxLenKey-len(k)) + v)
	}
	fmt.Println()
}

func (uc *UPnPClient) parseServices() []Service {
	services := make([]Service, 0)

	//	dr := newRequest("GET", uc.URL+IGDESCENDPOINT, "")
	dr := newRequest("GET", uc.URL+"/tr64desc.xml", "")
	decoder := xml.NewDecoder(do(dr))
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "service" {
				var service Service
				if err := decoder.DecodeElement(&service, &se); err != nil {
					panic(err)
				}

				//if strings.Contains(service.ServiceId, "WLANConfiguration") {
				service.Actions = uc.parseActions(service)
				services = append(services, service)
				//}

			}
		}
	}
	return services
}

func (uc *UPnPClient) parseActions(service Service) []Action {
	actions := make([]Action, 0)

	dr := newRequest("GET", uc.URL+service.SCPDURL, "")
	decoder := xml.NewDecoder(do(dr))
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "action" {
				var action Action
				if err := decoder.DecodeElement(&action, &se); err != nil {
					panic(err)
				}
				if IsActionGetOnly(action) {
					actions = append(actions, action)
				}
			}
		}
	}
	return actions
}

func IsActionGetOnly(action Action) bool {
	match, _ := regexp.MatchString("^(.*Get)+[A-z]*", action.Name)
	if !match {
		return false
	}
	for _, a := range action.Arguments {
		if a.Direction == "in" {
			return false
		}
	}
	return len(action.Arguments) > 0
}
