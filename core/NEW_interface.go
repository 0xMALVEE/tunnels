package core

import (
	"bytes"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackpal/gateway"
)

func PingConnections(MONITOR chan int) {
	defer func() {
		time.Sleep(30 * time.Second)
		MONITOR <- 3
	}()
	defer RecoverAndLogToFile()
	for _, v := range ConList {
		if v == nil {
			continue
		}

		var err error
		if v.EH != nil {
			out := v.EH.SEAL.Seal1([]byte{255, 255, 255, 255}, v.Index)
			_, err = v.Con.Write(out)
		}

		if time.Since(v.TunnelSTATS.PingTime).Seconds() > 30 || err != nil {
			if v.Meta.AutoReconnect {
				DEBUG("30+ Seconds since ping from ", v.Meta.Tag, " attempting reconnection")
				_, _ = PublicConnect(v.UICR)
			} else {
				DEBUG("30+ Seconds since ping from ", v.Meta.Tag, " disconnecting")
				_ = Disconnect(v.Meta.WindowsGUID, true, false)
			}
		}
	}
}

func getDefaultGatewayAndInterface() {
	defer RecoverAndLogToFile()
	var err error

	OLD_GATEWAY := make([]byte, 4)
	var NEW_GATEWAY net.IP
	copy(OLD_GATEWAY, DEFAULT_GATEWAY.To4())

	// DEFAULT_GATEWAY, err = FindGateway()
	NEW_GATEWAY, err = gateway.DiscoverGateway()
	if err != nil {
		DEBUG("default gateway not found", err)
		return
	}

	for _, v := range GLOBAL_STATE.ActiveConnections {
		if v.Tag == "DEFAULT" {
			return
		}
	}

	for _, v := range C.Connections {
		if v.IPv4Address == NEW_GATEWAY.To4().String() {
			// DEBUG(fmt.Sprintf("discovered gateway is the same as gateway for connection (%s)", v.Tag))
			return
		}
	}

	if bytes.Equal(OLD_GATEWAY, NEW_GATEWAY.To4()) {
		return
	}

	DEBUG("new gateway discovered", NEW_GATEWAY)
	DEFAULT_GATEWAY = NEW_GATEWAY

	DEFAULT_INTERFACE, err = gateway.DiscoverInterface()
	if err != nil {
		ERROR("could not find default interface", err)
		return
	} else {

		DNSClient.Dialer.LocalAddr = &net.UDPAddr{
			IP: DEFAULT_INTERFACE,
		}

		ifList, _ := net.Interfaces()

	LOOP:
		for _, v := range ifList {
			addrs, e := v.Addrs()
			if e != nil {
				continue
			}
			for _, iv := range addrs {
				if strings.Split(iv.String(), "/")[0] == DEFAULT_INTERFACE.To4().String() {
					DEFAULT_INTERFACE_ID = v.Index
					DEFAULT_INTERFACE_NAME = v.Name
					_ = GetDNSServers(strconv.Itoa(v.Index))
					break LOOP
				}
			}
		}

	}

	if DEFAULT_DNS_SERVERS == nil {
		if C.DNS2Default != "" {
			DEFAULT_DNS_SERVERS = []string{C.DNS1Default, C.DNS2Default}
		} else {
			DEFAULT_DNS_SERVERS = []string{C.DNS1Default}
		}
	}

	DEBUG(
		"Default interface",
		DEFAULT_INTERFACE_NAME,
		DEFAULT_INTERFACE_ID,
		DEFAULT_INTERFACE,
	)
}

func GetDefaultGateway(MONITOR chan int) {
	defer func() {
		if DEFAULT_GATEWAY != nil {
			time.Sleep(5 * time.Second)
		} else {
			time.Sleep(2 * time.Second)
		}

		MONITOR <- 4
	}()
	getDefaultGatewayAndInterface()
}
