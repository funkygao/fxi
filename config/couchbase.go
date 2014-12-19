package config

import (
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
)

type ConfigCouchbase struct {
	Servers []string
}

func (this *ConfigCouchbase) loadConfig(cf *conf.Conf) {
	this.Servers = cf.StringList("servers", nil)
	log.Debug("couchbase conf: %+v", *this)
}

func (this *ConfigCouchbase) Enabled() bool {
	return len(this.Servers) > 0
}
