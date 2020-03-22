package main

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

const ENDPOINT string = "/tr64desc.xml"

type UPnPClient struct {
	URL      string
	user     string
	password string
	// Services and actions (service is key, actions value) to fetch
	servicesActions map[string][]string
}

func NewUPnPClient(cfg *Config, servicesActions map[string][]string) *UPnPClient {
	return &UPnPClient{
		URL:             fmt.Sprintf("http://%s:49000", cfg.URL),
		user:            cfg.User,
		password:        cfg.Password,
		servicesActions: servicesActions,
	}
}

type serviceActionValue struct {
	serviceType string
	actionName  string
	variable    string
	value       string
}

func (uc *UPnPClient) Execute() []serviceActionValue {
	var result []serviceActionValue
	for _, service := range uc.parseServices() {
		serviceToFetch := len(uc.servicesActions) == 0
		var actionsToFetch []string
		for k, actions := range uc.servicesActions {
			if strings.Contains(service.ServiceType, k) {
				actionsToFetch = actions
				serviceToFetch = true
				break
			}
		}

		if serviceToFetch {
			for _, action := range uc.parseActions(service) {
				actionToFetch := len(uc.servicesActions) == 0
				for _, a := range actionsToFetch {
					if a == action.Name {
						actionToFetch = true
						break
					}
				}
				if actionToFetch {

					message := fmt.Sprintf(`
		<?xml version="1.0"?> 
        <s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" 
				s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"> 
            <s:Body><u:%s xmlns:u='%s'/></s:Body>
		</s:Envelope>`, action.Name, service.ServiceType)

					dr := newRequest("POST", uc.URL+service.ControlURL, message)

					dr.Header.Add("Content-Type", "text/xml")
					dr.Header.Add("charset", "utf-8")
					dr.Header.Add("SoapAction", fmt.Sprintf("%s#%s", service.ServiceType, action.Name))

					content := do(dr, uc.user, uc.password)
					defer content.Close()
					decoder := xml.NewDecoder(content)
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
										result = append(result, serviceActionValue{
											serviceType: service.ServiceType,
											actionName:  action.Name,
											variable:    argument.RelatedStateVariable,
											value:       string(element),
										})
									}
								}
							}
						}
					}
				}
			}
		}
	}
	printResult(result)
	return result
}

func printResult(m []serviceActionValue) {
	for _, s := range m {
		log.Debugf("%s:::%s/%s   =   %s\n", s.serviceType, s.actionName, s.variable, s.value)
	}
}

func (uc *UPnPClient) parseServices() []Service {
	services := make([]Service, 0)

	dr := newRequest("GET", uc.URL+ENDPOINT, "")

	decoder := xml.NewDecoder(do(dr, uc.user, uc.password))
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
	decoder := xml.NewDecoder(do(dr, uc.user, uc.password))
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
