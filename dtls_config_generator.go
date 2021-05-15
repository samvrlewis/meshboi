package meshboi

import (
	"github.com/pion/dtls/v2"
	"inet.af/netaddr"
)

func getDtlsConfig(vpnIp netaddr.IP, psk []byte) *dtls.Config {
	return &dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			return psk, nil
		},
		// We set the PSK identity hint as the IP address of this member in the
		// VPN as an quick and hacky way of signalling (out of band) who this
		// member is to other members we connect to. A more robust way of
		// achieving this would be to define an OOB messaging scheme to do this
		// with instead.
		PSKIdentityHint:      []byte(vpnIp.String()),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_128_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	}
}
