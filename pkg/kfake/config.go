package kfake

import "time"

// Opt is an option to configure a client.
type Opt interface {
	apply(*cfg)
}

type opt struct{ fn func(*cfg) }

func (opt opt) apply(cfg *cfg) { opt.fn(cfg) }

type cfg struct {
	nbrokers        int
	ports           []int
	logger          Logger
	clusterID       string
	allowAutoTopic  bool
	defaultNumParts int

	minSessionTimeout time.Duration
	maxSessionTimeout time.Duration

	enableSASL bool
	sasls      map[struct{ m, u string }]string // cleared after client initialization
}

// NumBrokers sets the number of brokers to start in the fake cluster.
func NumBrokers(n int) Opt {
	return opt{func(cfg *cfg) { cfg.nbrokers = n }}
}

// Ports sets the ports to listen on, overriding randomly choosing NumBrokers
// amount of ports.
func Ports(ports ...int) Opt {
	return opt{func(cfg *cfg) { cfg.ports = ports }}
}

// WithLogger sets the logger to use.
func WithLogger(logger Logger) Opt {
	return opt{func(cfg *cfg) { cfg.logger = logger }}
}

// ClusterID sets the cluster ID to return in metadata responses.
func ClusterID(clusterID string) Opt {
	return opt{func(cfg *cfg) { cfg.clusterID = clusterID }}
}

// AllowAutoTopicCreation allows metadata requests to create topics if the
// metadata request has its AllowAutoTopicCreation field set to true.
func AllowAutoTopicCreation() Opt {
	return opt{func(cfg *cfg) { cfg.allowAutoTopic = true }}
}

// DefaultNumPartitions sets the number of partitions to create by default for
// auto created topics / CreateTopics with -1 partitions.
func DefaultNumPartitions(n int) Opt {
	return opt{func(cfg *cfg) { cfg.defaultNumParts = n }}
}

// GroupMinSessionTimeout sets the cluster's minimum session timeout allowed
// for groups, overriding the default 6 seconds.
func GroupMinSessionTimeout(d time.Duration) Opt {
	return opt{func(cfg *cfg) { cfg.minSessionTimeout = d }}
}

// GroupMaxSessionTimeout sets the cluster's maximum session timeout allowed
// for groups, overriding the default 5 minutes.
func GroupMaxSessionTimeout(d time.Duration) Opt {
	return opt{func(cfg *cfg) { cfg.maxSessionTimeout = d }}
}

// EnableSASL enables SASL authentication for the cluster. If you do not
// configure a bootstrap user / pass, the default superuser is "admin" /
// "admin" with the SCRAM-SHA-256 SASL mechanisms.
func EnableSASL() Opt {
	return opt{func(cfg *cfg) { cfg.enableSASL = true }}
}

// Superuser seeds the cluster with a superuser. The method must be either
// PLAIN, SCRAM-SHA-256, or SCRAM-SHA-512.
// Note that PLAIN superusers cannot be deleted.
// SCRAM superusers can be modified with AlterUserScramCredentials.
// If you delete all SASL users, the kfake cluster will be unusable.
func Superuser(method, user, pass string) Opt {
	return opt{func(cfg *cfg) { cfg.sasls[struct{ m, u string }{method, user}] = pass }}
}
