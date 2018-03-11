package go_i2cp

import (
	"strconv"
	"strings"
)

type Version struct {
	major, minor, micro, qualifier uint16
	version                        string
}

func parseVersion(str string) Version {
	var v = Version{}
	segments := strings.Split(str, ".")
	n := len(segments)
	if n > 0 {
		i, _ := strconv.Atoi(segments[0])
		v.major = uint16(i)
	}
	if n > 1 {
		i, _ := strconv.Atoi(segments[1])
		v.minor = uint16(i)
	}
	if n > 2 {
		i, _ := strconv.Atoi(segments[2])
		v.micro = uint16(i)
	}
	if n > 3 {
		i, _ := strconv.Atoi(segments[3])
		v.qualifier = uint16(i)
	}
	return v
}

func (v *Version) compare(other Version) int {
	if v.major != other.major {
		if (v.major - other.major) > 0 {
			return 1
		} else {
			return -1
		}
	}
	if v.minor != other.minor {
		if (v.minor - other.minor) > 0 {
			return 1
		} else {
			return -1
		}
	}
	if v.micro != other.micro {
		if (v.micro - other.micro) > 0 {
			return 1
		} else {
			return -1
		}
	}
	if v.qualifier != other.qualifier {
		if (v.qualifier - other.qualifier) > 0 {
			return 1
		} else {
			return -1
		}
	}
	return 0
}
