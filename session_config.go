package go_i2cp

import (
	"bufio"
	"os"
	"regexp"
	"time"
)

type SessionConfigProperty int

const (
	SESSION_CONFIG_PROP_CRYPTO_LOW_TAG_THRESHOLD SessionConfigProperty = iota
	SESSION_CONFIG_PROP_CRYPTO_TAGS_TO_SEND

	SESSION_CONFIG_PROP_I2CP_DONT_PUBLISH_LEASE_SET
	SESSION_CONFIG_PROP_I2CP_FAST_RECEIVE
	SESSION_CONFIG_PROP_I2CP_GZIP
	SESSION_CONFIG_PROP_I2CP_MESSAGE_RELIABILITY
	SESSION_CONFIG_PROP_I2CP_PASSWORD
	SESSION_CONFIG_PROP_I2CP_USERNAME

	SESSION_CONFIG_PROP_INBOUND_ALLOW_ZERO_HOP
	SESSION_CONFIG_PROP_INBOUND_BACKUP_QUANTITY
	SESSION_CONFIG_PROP_INBOUND_IP_RESTRICTION
	SESSION_CONFIG_PROP_INBOUND_LENGTH
	SESSION_CONFIG_PROP_INBOUND_LENGTH_VARIANCE
	SESSION_CONFIG_PROP_INBOUND_NICKNAME
	SESSION_CONFIG_PROP_INBOUND_QUANTITY

	SESSION_CONFIG_PROP_OUTBOUND_ALLOW_ZERO_HOP
	SESSION_CONFIG_PROP_OUTBOUND_BACKUP_QUANTITY
	SESSION_CONFIG_PROP_OUTBOUND_IP_RESTRICTION
	SESSION_CONFIG_PROP_OUTBOUND_LENGTH
	SESSION_CONFIG_PROP_OUTBOUND_LENGTH_VARIANCE
	SESSION_CONFIG_PROP_OUTBOUND_NICKNAME
	SESSION_CONFIG_PROP_OUTBOUND_PRIORITY
	SESSION_CONFIG_PROP_OUTBOUND_QUANTITY

	NR_OF_SESSION_CONFIG_PROPERTIES
)

var sessionOptions = [NR_OF_SESSION_CONFIG_PROPERTIES]string{
	"crypto.lowTagThreshold",
	"crypto.tagsToSend",
	"i2cp.dontPublishLeaseSet",
	"i2cp.fastReceive",
	"i2cp.gzip",
	"i2cp.messageReliability",
	"i2cp.password",
	"i2cp.username",

	"inbound.allowZeroHop",
	"inbound.backupQuantity",
	"inbound.IPRestriction",
	"inbound.length",
	"inbound.lengthVariance",
	"inbound.nickname",
	"inbound.quantity",

	"outbound.allowZeroHop",
	"outbound.backupQuantity",
	"outbound.IPRestriction",
	"outbound.length",
	"outbound.lengthVariance",
	"outbound.nickname",
	"outbound.priority",
	"outbound.quantity",
}
var configRegex = regexp.MustCompile("\\s*([\\w.]+)=\\s*(.+)\\s*;\\s*")

type SessionConfig struct {
	properties  [NR_OF_SESSION_CONFIG_PROPERTIES]string
	date        uint64
	destination *Destination
}

func NewSessionConfigFromDestinationFile(filename string) (config SessionConfig) {
	var home string
	if file, err := os.Open(filename); err == nil {
		dest, err := NewDestinationFromFile(file)
		config.destination = &dest
		if err != nil {
			Warning(SESSION_CONFIG, "Failed to load destination from file '%s', a new destination will be generated.", filename)
		}
	}
	if config.destination == nil {
		dest, err := NewDestination()
		config.destination = &dest
	}
	if len(filename) > 0 {
		config.destination.WriteToFile(filename)
	}
	home = os.Getenv("HOME")
	if len(home) > 0 {
		configFile := home + "/.i2cp.conf"
		ParseConfig(configFile, func(name, value string) {
			if prop := config.propFromString(name); prop >= 0 {
				config.SetProperty(prop, value)
			}
		})
	}
	return config
}
func (config *SessionConfig) writeToMessage(stream *Stream) {
	config.destination.WriteToMessage(stream)
	config.writeMappingToMessage(stream)
	stream.WriteUint64(uint64(time.Now().Unix()))
	GetCryptoInstance().WriteSignatureToStream(&config.destination.sgk, stream)
}
func (config *SessionConfig) writeMappingToMessage(stream *Stream) (err error) {
	is := NewStream(make([]byte, 0xffff))
	count := 0
	for i := 0; i < int(NR_OF_SESSION_CONFIG_PROPERTIES); i++ {
		var option string
		if sc.properties[i] == "" {
			continue
		}
		option = sc.configOptLookup(SessionConfigProperty(i))
		if option == "" {
			continue
		}
		is.Write([]byte(option + "=" + sc.properties[i] + ";"))
		count++
	}
	Debug(SESSION_CONFIG, "Writing %d options to mapping table", count)
	err = stream.WriteUint16(uint16(is.Len()))
	if is.Len() > 0 {
		_, err = stream.Write(is.Bytes())
	}
	return
}
func (config *SessionConfig) configOptLookup(property SessionConfigProperty) string {
	return sessionOptions[property]
}
func (config *SessionConfig) propFromString(name string) SessionConfigProperty {
	for i := 0; SessionConfigProperty(i) < NR_OF_SESSION_CONFIG_PROPERTIES; i++ {
		if sessionOptions[i] == name {
			return SessionConfigProperty(i)
		}
	}
	return SessionConfigProperty(-1)
}
func (config *SessionConfig) SetProperty(prop SessionConfigProperty, value string) {
	config.properties[prop] = value
}
func ParseConfig(s string, cb func(string, string)) {
	file, err := os.Open(s)
	if err != nil {
		Error(SESSION_CONFIG, err.Error())
		return
	}
	Debug(SESSION_CONFIG, "Parsing config file '%s'", s)
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		line := scan.Text()
		groups := configRegex.FindStringSubmatch(line)
		if len(groups) != 3 {
			continue
		}
		cb(groups[1], groups[2])
	}
	if err := scan.Err(); err != nil {
		Error(SESSION_CONFIG, "reading input from %s config %s", s, err.Error())
	}
}
