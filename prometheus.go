package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type FritzBoxCollector struct {
	Config                                               *Config
	wanTotalBytesSentOffset, wanTotalBytesReceivedOffset float64
	lastWanTotalBytesSent, lastWanTotalBytesReceived     float64
}

func newFritzBoxCollector(config *Config) *FritzBoxCollector {
	return &FritzBoxCollector{
		Config: config,
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

	wanTotalBytesReceived := extract(filterByService(values, "WANCommonInterfaceConfig", "GetTotalBytesReceived", "TotalBytesReceived"))

	if wanTotalBytesReceived < collector.lastWanTotalBytesReceived {
		collector.wanTotalBytesReceivedOffset = wanTotalBytesReceived
	}
	wanTotalBytesReceivedCorrected := wanTotalBytesReceived - collector.wanTotalBytesReceivedOffset
	collector.lastWanTotalBytesReceived = wanTotalBytesReceived

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_bytes_received",
		"WAN total bytes received",
		nil,
		nil,
	), prometheus.CounterValue, wanTotalBytesReceived)

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_bytes_received_corrected",
		"WAN total bytes received corrected",
		nil,
		nil,
	), prometheus.CounterValue, wanTotalBytesReceivedCorrected)

	wanTotalBytesSent := extract(filterByService(values, "WANCommonInterfaceConfig", "GetTotalBytesSent", "TotalBytesSent"))

	if wanTotalBytesSent < collector.lastWanTotalBytesSent {
		collector.wanTotalBytesSentOffset = wanTotalBytesSent
	}
	wanTotalBytesSentCorrected := wanTotalBytesSent - collector.wanTotalBytesSentOffset
	collector.lastWanTotalBytesSent = wanTotalBytesSent

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_bytes_sent",
		"WAN total bytes sent",
		nil,
		nil,
	), prometheus.CounterValue, wanTotalBytesSent)

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_bytes_sent_corrected",
		"WAN total bytes sent corrected",
		nil,
		nil,
	), prometheus.CounterValue, wanTotalBytesSentCorrected)

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_packets_received",
		"WAN total packets received",
		nil,
		nil,
	), prometheus.CounterValue, extract(filterByService(values, "WANCommonInterfaceConfig", "GetTotalPacketsReceived", "TotalPacketsReceived")))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_wan_total_packets_sent",
		"WAN total packets sent",
		nil,
		nil,
	), prometheus.CounterValue, extract(filterByService(values, "WANCommonInterfaceConfig", "GetTotalPacketsSent", "TotalPacketsSent")))

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
	), prometheus.CounterValue, extract(filterByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.BytesReceived")))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_lan_eth_total_bytes_sent",
		"LAN ethernet total bytes sent",
		nil,
		nil,
	), prometheus.CounterValue, extract(filterByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.BytesSent")))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_lan_eth_total_packets_received",
		"LAN ethernet total packets received",
		nil,
		nil,
	), prometheus.CounterValue, extract(filterByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.PacketsReceived")))

	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
		"fb_lan_eth_total_packets_sent",
		"LAN ethernet total packets sent",
		nil,
		nil,
	), prometheus.CounterValue, extract(filterByService(values, "LANEthernetInterfaceConfig", "GetStatistics", "Stats.PacketsSent")))

	for i := 1; i <= 3; i++ {
		wlanName := filterByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetInfo", "SSID")
		wlanStandard := filterByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetInfo", "Standard")
		wlanNameStandard := fmt.Sprintf("%s (%s)", wlanName, wlanStandard)
		totalAssociations := filterByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetTotalAssociations", "TotalAssociations")
		totalPacketsSent := filterByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetStatistics", "TotalPacketsSent")
		totalPacketsReceived := filterByService(values, fmt.Sprintf("WLANConfiguration:%d", i), "GetStatistics", "TotalPacketsReceived")

		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
			"fb_wlan_number_associations",
			"Number of WLAN clients",
			[]string{"ssid_standard"},
			nil,
		), prometheus.GaugeValue, extract(totalAssociations), wlanNameStandard)

		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
			"fb_wlan_total_packets_sent",
			"WLAN total packets sent",
			[]string{"ssid_standard"},
			nil,
		), prometheus.CounterValue, extract(totalPacketsSent), wlanNameStandard)

		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
			"fb_wlan_total_packets_received",
			"WLAN total packets received",
			[]string{"ssid_standard"},
			nil,
		), prometheus.CounterValue, extract(totalPacketsReceived), wlanNameStandard)
	}

}
