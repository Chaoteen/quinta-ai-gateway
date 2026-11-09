package service

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/Chaoteen/quinta-ai-gateway/setting/system_setting"
)

func productionOutboundSecurity() bool {
	return strings.EqualFold(os.Getenv("APP_ENV"), "production")
}

// Connect to the validated IP itself, so a second DNS resolution cannot redirect
// a checked public hostname to IMDS, loopback or another private destination.
func dialOutbound(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	if !productionOutboundSecurity() {
		return dialer.DialContext(ctx, network, address)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid outbound address")
	}
	policy := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting("https://"+address, true, false,
		policy.DomainFilterMode, policy.IpFilterMode, policy.DomainList, policy.IpList, policy.AllowedPorts, true); err != nil {
		return nil, errors.New("outbound destination blocked by policy")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("outbound DNS resolution failed")
	}
	filter := common.SSRFProtection{AllowPrivateIp: false, IpFilterMode: policy.IpFilterMode, IpList: policy.IpList}
	for _, ip := range ips {
		if !filter.IsIPAccessAllowed(ip.IP) {
			return nil, errors.New("outbound resolved address blocked by policy")
		}
	}
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, errors.New("outbound connection failed")
}
