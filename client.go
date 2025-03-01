package neo4jorm

import (
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Config struct {
	URI      string
	Username string
	Password string
	Database string
	Debug    bool
}

type Client struct {
	driver neo4j.Driver
	config *Config
	debug  bool
}

func NewClient(config *Config) (*Client, error) {
	driver, err := neo4j.NewDriver(
		config.URI,
		neo4j.BasicAuth(config.Username, config.Password, ""),
	)
	if err != nil {
		return nil, err
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	if err = driver.VerifyConnectivity(); err != nil {
		return nil, err
	}

	return &Client{
		driver: driver,
		config: config,
		debug:  config.Debug,
	}, nil
}

func (c *Client) Model(model interface{}) *Model {
	return newModel(c, model)
}

func (c *Client) Close() error {
	return c.driver.Close()
}

// 事务支持
type Transaction struct {
	session neo4j.Session
	tx      neo4j.Transaction
}

func (c *Client) BeginTx() (*Transaction, error) {
	session := c.driver.NewSession(neo4j.SessionConfig{
		DatabaseName: c.config.Database,
	})
	tx, err := session.BeginTransaction()
	if err != nil {
		session.Close()
		return nil, err
	}
	return &Transaction{session: session, tx: tx}, nil
}

func (t *Transaction) Commit() error {
	defer t.session.Close()
	return t.tx.Commit()
}

func (t *Transaction) Rollback() error {
	defer t.session.Close()
	return t.tx.Rollback()
}
