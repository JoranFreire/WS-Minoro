package writer

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

type ClickRecord struct {
	LinkID         string
	TenantID       string
	ShortCode      string
	DestinationURL string
	IPHash         string
	Country        string
	City           string
	DeviceType     string
	Browser        string
	OS             string
	Referer        string
	Timestamp      time.Time
}

type CassandraWriter struct {
	session  *gocql.Session
	keyspace string
}

func NewCassandraWriter(hosts []string, keyspace string) *CassandraWriter {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 5 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		panic("failed to connect to cassandra: " + err.Error())
	}
	return &CassandraWriter{session: session, keyspace: keyspace}
}

func (w *CassandraWriter) WriteClick(ctx context.Context, click ClickRecord) error {
	linkID, err := uuid.Parse(click.LinkID)
	if err != nil {
		return err
	}
	tenantID, err := uuid.Parse(click.TenantID)
	if err != nil {
		return err
	}

	day := click.Timestamp.Format("2006-01-02")
	clickID := gocql.TimeUUID()

	return w.session.Query(`
		INSERT INTO clicks_by_link
		(link_id, day, click_id, destination_url, country, city,
		 device_type, browser, os, referer, ip_hash, tenant_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		linkID, day, clickID, click.DestinationURL, click.Country, click.City,
		click.DeviceType, click.Browser, click.OS, click.Referer, click.IPHash, tenantID,
	).WithContext(ctx).Exec()
}
