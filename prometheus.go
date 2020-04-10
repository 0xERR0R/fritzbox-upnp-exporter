package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type FritzBoxCollector struct {
	Config     *Config
	lastValues map[string]float64
	offsets    map[string]float64
}

func newFritzBoxCollector(config *Config) *FritzBoxCollector {
	return &FritzBoxCollector{
		Config:     config,
		lastValues: make(map[string]float64),
		offsets:    make(map[string]float64),
	}
}

func (collector *FritzBoxCollector) Describe(ch chan<- *prometheus.Desc) {

}

func filterByService(in []serviceActionValue, service string, action string, variable string) string {
	for _, a := range in {
		if strings.Contains(a.serviceType, service) && a.actionName == action && a.variable == variable {
			return a.value
		}
	}
	log.Warnf("value for service %s, action %s, variable %s not found", service, action, variable)
	return ""
}

func filterConvertByService(in []serviceActionValue, service string, action string, variable string) float64 {
	return extract(filterByService(in, service, action, variable))
}

func (collector *FritzBoxCollector) filterConvertAndCorrectByService(in []serviceActionValue, service string, action string, variable string) float64 {
	newValue := filterConvertByService(in, service, action, variable)

	key := fmt.Sprintf("%s/%s/%s", service, action, variable)

	lastValue := collector.lastValues[key]

	if newValue < lastValue {
		collector.offsets[key] = newValue
	}
	correctedValue := newValue - collector.offsets[key]
	collector.lastValues[key] = newValue

	return correctedValue
}

func extract(val string) float64 {
	if s, err := strconv.ParseFloat(val, 64); err == nil {
		return s
	}
	return 0.0
}

func (collector *FritzBoxCollector) Collect(ch chan<- prometheus.Metric) {
	uPnPClient := NewUPnPClient(
		collector.Config,
		map[string][]string{
			"WANCommonInterfaceConfig":   {"GetTotalBytesReceived", "GetTotalBytesSent", "GetTotalPacketsSent", "GetTotalPacketsReceived"},
			"WANPPPConnection":           {"GetExternalIPAddress", "GetStatusInfo"},
			"LANEthernetInterfaceConfig": {"GetStatistics"},
			"WLANConfiguration":          {"GetInfo", "GetTotalAssociations", "GetStatistics"},
		},
	)
	values := uPnPClient.Execute()

	wanTotalBytesReceived := collector.filterConvertAndCorrectByService(values, "WANCommonInterfaceConfig", "GetTotalBytesReceived", "TotalBytesReceived")

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_bytes_received",
		"WAN total bytes received",
		nil,
		nil,
	), prometheus.CounterValue, wanTotalBytesReceived)

	wanTotalBytesSent := collector.filterConvertAndCorrectByService(values, "WANCommonInterfaceConfig", "GetTotalBytesSent", "TotalBytesSent")

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_bytes_sent",
		"WAN total bytes sent",
		nil,
		nil,
	), prometheus.CounterValue, wanTotalBytesSent)

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_packets_received",
		"WAN total packets received",
		nil,
		nil,
	), prometheus.CounterValue, collector.filterConvertAndCorrectByService(values, "WANCommonInterfaceConfig", "GetTotalPacketsReceived", "TotalPacketsReceived"))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_packets_sent",
		"WAN total packets sent",
		nil,
		nil,
	), prometheus.CounterValue, collector.filterConvertAndCorrectByService(values, "WANCommonInterfaceConfig", "GetTotalPacketsSent", "TotalPacketsSent"))

	externalIP := filterByService(values, "WANPPPConnection", "GetExternalIPAddress", "ExternalIPAddress")
	connectionStatus := filterByService(values, "WANPPPConnection", "GetStatusInfo", "ConnectionStatus")
	uptime := filterByService(values, "WANPPPConnection", "GetStatusInfo", "Uptime")
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wanppp_status_uptime",
		"WAN PPP uptime",
		[]string{"ip", "status"},
		nil,
	), prometheus.CounterValue, extract(uptime), externalIP, connectionStatus)

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_lan_eth_total_bytes_received",
		"LAN ethernet total bytes received",
		nil,
		nil,
	), prometheus.CounterValue, collector.filterConvertAndCorrectByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.BytesReceived"))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_lan_eth_total_bytes_sent",
		"LAN ethernet total bytes sent",
		nil,
		nil,
	), prometheus.CounterValue, collector.filterConvertAndCorrectByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.BytesSent"))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_lan_eth_total_packets_received",
		"LAN ethernet total packets received",
		nil,
		nil,
	), prometheus.CounterValue, collector.filterConvertAndCorrectByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.PacketsReceived"))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_lan_eth_total_packets_sent",
		"LAN ethernet total packets sent",
		nil,
		nil,
	), prometheus.CounterValue, collector.filterConvertAndCorrectByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.PacketsSent"))

	for i := 1; i <= 3; i++ {
		wlanName := filterByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetInfo", "SSID")
		wlanStandard := filterByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetInfo", "Standard")
		wlanNameStandard := fmt.Sprintf("%d:%s (%s)", i, wlanName, wlanStandard)
		totalAssociations := filterConvertByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetTotalAssociations", "TotalAssociations")
		totalPacketsSent := collector.filterConvertAndCorrectByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetStatistics", "TotalPacketsSent")
		totalPacketsReceived := collector.filterConvertAndCorrectByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetStatistics", "TotalPacketsReceived")

		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
			"fb_wlan_number_associations",
			"Number of WLAN clients",
			[]string{"ssid_standard"},
			nil,
		), prometheus.GaugeValue, totalAssociations, wlanNameStandard)

		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
			"fb_wlan_total_packets_sent",
			"WLAN total packets sent",
			[]string{"ssid_standard"},
			nil,
		), prometheus.CounterValue, totalPacketsSent, wlanNameStandard)

		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
			"fb_wlan_total_packets_received",
			"WLAN total packets received",
			[]string{"ssid_standard"},
			nil,
		), prometheus.CounterValue, totalPacketsReceived, wlanNameStandard)
	}

}
