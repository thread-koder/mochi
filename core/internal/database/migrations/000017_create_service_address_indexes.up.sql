CREATE TABLE IF NOT EXISTS service_ips (
    ip VARCHAR(50) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    PRIMARY KEY (ip, namespace, name),
    FOREIGN KEY (namespace, name) REFERENCES services(namespace, name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS node_ips (
    ip VARCHAR(50) NOT NULL PRIMARY KEY,
    node_name VARCHAR(255) NOT NULL REFERENCES nodes(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_node_ips_node_name ON node_ips(node_name);

CREATE TABLE IF NOT EXISTS service_node_ports (
    protocol VARCHAR(50) NOT NULL,
    port INT NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    PRIMARY KEY (protocol, port, namespace, name),
    FOREIGN KEY (namespace, name) REFERENCES services(namespace, name) ON DELETE CASCADE
);
