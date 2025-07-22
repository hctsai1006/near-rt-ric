-- Database initialization script for O-RAN Near-RT RIC
-- Creates necessary tables and initial data for the RIC platform

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create schemas for different components
CREATE SCHEMA IF NOT EXISTS e2;
CREATE SCHEMA IF NOT EXISTS a1;
CREATE SCHEMA IF NOT EXISTS o1;
CREATE SCHEMA IF NOT EXISTS xapp;
CREATE SCHEMA IF NOT EXISTS monitoring;

-- Set default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA e2 GRANT ALL ON TABLES TO ric_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA a1 GRANT ALL ON TABLES TO ric_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA o1 GRANT ALL ON TABLES TO ric_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA xapp GRANT ALL ON TABLES TO ric_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA monitoring GRANT ALL ON TABLES TO ric_user;

-- Grant schema usage
GRANT USAGE ON SCHEMA e2 TO ric_user;
GRANT USAGE ON SCHEMA a1 TO ric_user;
GRANT USAGE ON SCHEMA o1 TO ric_user;
GRANT USAGE ON SCHEMA xapp TO ric_user;
GRANT USAGE ON SCHEMA monitoring TO ric_user;

-- E2 Interface Tables
CREATE TABLE IF NOT EXISTS e2.e2_nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id VARCHAR(255) UNIQUE NOT NULL,
    node_type VARCHAR(50) NOT NULL, -- 'enb', 'gnb', 'du', 'cu'
    plmn_id VARCHAR(6) NOT NULL,
    nb_id BIGINT NOT NULL,
    ran_functions JSONB,
    connection_status VARCHAR(20) DEFAULT 'disconnected',
    connection_time TIMESTAMP,
    last_seen TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS e2.e2_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscription_id VARCHAR(255) UNIQUE NOT NULL,
    node_id VARCHAR(255) NOT NULL REFERENCES e2.e2_nodes(node_id),
    ran_function_id INTEGER NOT NULL,
    request_id INTEGER NOT NULL,
    instance_id INTEGER NOT NULL,
    subscription_details JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS e2.e2_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_type VARCHAR(50) NOT NULL,
    node_id VARCHAR(255),
    subscription_id VARCHAR(255),
    procedure_code INTEGER,
    message_data JSONB,
    direction VARCHAR(10) NOT NULL, -- 'inbound', 'outbound'
    timestamp TIMESTAMP DEFAULT NOW(),
    processing_time_ms INTEGER
);

-- A1 Interface Tables  
CREATE TABLE IF NOT EXISTS a1.a1_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id VARCHAR(255) UNIQUE NOT NULL,
    policy_type_id VARCHAR(255) NOT NULL,
    policy_name VARCHAR(255),
    policy_description TEXT,
    policy_schema JSONB,
    policy_data JSONB NOT NULL,
    enforcement_status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS a1.a1_policy_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_type_id VARCHAR(255) UNIQUE NOT NULL,
    policy_type_name VARCHAR(255) NOT NULL,
    policy_type_description TEXT,
    schema JSONB NOT NULL,
    version VARCHAR(20) DEFAULT '1.0',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS a1.a1_enrichment_info (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    info_type_id VARCHAR(255) NOT NULL,
    info_producer_id VARCHAR(255) NOT NULL,
    info_job_id VARCHAR(255) NOT NULL,
    info_data JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(info_type_id, info_producer_id, info_job_id)
);

-- O1 Interface Tables
CREATE TABLE IF NOT EXISTS o1.o1_alarms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alarm_id VARCHAR(255) UNIQUE NOT NULL,
    alarm_type VARCHAR(100) NOT NULL,
    managed_object VARCHAR(255) NOT NULL,
    source VARCHAR(255) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    probable_cause VARCHAR(255),
    specific_problem TEXT,
    additional_text TEXT,
    raised_time TIMESTAMP NOT NULL,
    cleared_time TIMESTAMP,
    acknowledged_time TIMESTAMP,
    acknowledged_by VARCHAR(255),
    additional_info JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS o1.o1_configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    change_id VARCHAR(255) UNIQUE NOT NULL,
    session_id INTEGER NOT NULL,
    username VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP DEFAULT NOW(),
    operation VARCHAR(50) NOT NULL,
    datastore VARCHAR(50) NOT NULL,
    config_path VARCHAR(1000),
    old_value JSONB,
    new_value JSONB,
    status VARCHAR(20) DEFAULT 'success',
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS o1.o1_performance_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_id VARCHAR(255) UNIQUE NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    managed_object VARCHAR(255) NOT NULL,
    metric_type VARCHAR(50) NOT NULL,
    value NUMERIC,
    unit VARCHAR(50),
    timestamp TIMESTAMP NOT NULL,
    collection_time TIMESTAMP DEFAULT NOW(),
    granularity INTERVAL,
    labels JSONB,
    metadata JSONB
);

-- xApp Framework Tables
CREATE TABLE IF NOT EXISTS xapp.xapps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    xapp_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    vendor VARCHAR(255),
    license VARCHAR(100),
    deployment_spec JSONB NOT NULL,
    resource_requirements JSONB,
    interfaces JSONB,
    dependencies JSONB,
    config_schema JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS xapp.xapp_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    instance_id VARCHAR(255) UNIQUE NOT NULL,
    xapp_id VARCHAR(255) NOT NULL REFERENCES xapp.xapps(xapp_id),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    lifecycle_state VARCHAR(20) DEFAULT 'created',
    config JSONB,
    runtime_info JSONB,
    health_status JSONB,
    metrics JSONB,
    last_error TEXT,
    error_count INTEGER DEFAULT 0,
    deployment_target VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    stopped_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS xapp.xapp_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id VARCHAR(255) UNIQUE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    xapp_id VARCHAR(255),
    instance_id VARCHAR(255),
    timestamp TIMESTAMP DEFAULT NOW(),
    source VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    details JSONB,
    severity VARCHAR(20) DEFAULT 'info'
);

-- Monitoring Tables
CREATE TABLE IF NOT EXISTS monitoring.metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(255) NOT NULL,
    metric_type VARCHAR(50) NOT NULL, -- 'counter', 'gauge', 'histogram'
    component VARCHAR(100) NOT NULL,
    instance VARCHAR(255),
    labels JSONB,
    value NUMERIC NOT NULL,
    timestamp TIMESTAMP DEFAULT NOW(),
    INDEX(metric_name, component, timestamp)
);

CREATE TABLE IF NOT EXISTS monitoring.health_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component VARCHAR(100) NOT NULL,
    instance VARCHAR(255),
    status VARCHAR(20) NOT NULL, -- 'healthy', 'unhealthy', 'unknown'
    message TEXT,
    response_time_ms INTEGER,
    timestamp TIMESTAMP DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_e2_nodes_status ON e2.e2_nodes(connection_status);
CREATE INDEX IF NOT EXISTS idx_e2_nodes_last_seen ON e2.e2_nodes(last_seen);
CREATE INDEX IF NOT EXISTS idx_e2_subscriptions_node_id ON e2.e2_subscriptions(node_id);
CREATE INDEX IF NOT EXISTS idx_e2_subscriptions_status ON e2.e2_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_e2_messages_timestamp ON e2.e2_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_e2_messages_node_id ON e2.e2_messages(node_id);

CREATE INDEX IF NOT EXISTS idx_a1_policies_type ON a1.a1_policies(policy_type_id);
CREATE INDEX IF NOT EXISTS idx_a1_policies_status ON a1.a1_policies(enforcement_status);
CREATE INDEX IF NOT EXISTS idx_a1_enrichment_info_type ON a1.a1_enrichment_info(info_type_id);

CREATE INDEX IF NOT EXISTS idx_o1_alarms_status ON o1.o1_alarms(status);
CREATE INDEX IF NOT EXISTS idx_o1_alarms_severity ON o1.o1_alarms(severity);
CREATE INDEX IF NOT EXISTS idx_o1_alarms_raised_time ON o1.o1_alarms(raised_time);
CREATE INDEX IF NOT EXISTS idx_o1_configurations_timestamp ON o1.o1_configurations(timestamp);
CREATE INDEX IF NOT EXISTS idx_o1_performance_metrics_timestamp ON o1.o1_performance_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_o1_performance_metrics_name ON o1.o1_performance_metrics(metric_name);

CREATE INDEX IF NOT EXISTS idx_xapps_type ON xapp.xapps(type);
CREATE INDEX IF NOT EXISTS idx_xapp_instances_status ON xapp.xapp_instances(status);
CREATE INDEX IF NOT EXISTS idx_xapp_instances_xapp_id ON xapp.xapp_instances(xapp_id);
CREATE INDEX IF NOT EXISTS idx_xapp_events_timestamp ON xapp.xapp_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_xapp_events_type ON xapp.xapp_events(event_type);

CREATE INDEX IF NOT EXISTS idx_metrics_name_component ON monitoring.metrics(metric_name, component);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON monitoring.metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_health_checks_component ON monitoring.health_checks(component);

-- Insert initial data
INSERT INTO a1.a1_policy_types (policy_type_id, policy_type_name, policy_type_description, schema) VALUES 
('qos-policy-type', 'QoS Policy', 'Quality of Service policy for RAN slicing', 
 '{"type": "object", "properties": {"slice_id": {"type": "string"}, "qos_parameters": {"type": "object"}}}'),
('rrc-policy-type', 'RRC Policy', 'Radio Resource Control policy', 
 '{"type": "object", "properties": {"cell_id": {"type": "string"}, "rrc_parameters": {"type": "object"}}}'),
('mobility-policy-type', 'Mobility Management Policy', 'Handover and mobility optimization policy',
 '{"type": "object", "properties": {"handover_threshold": {"type": "number"}, "mobility_parameters": {"type": "object"}}}')
ON CONFLICT (policy_type_id) DO NOTHING;

-- Create triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to relevant tables
CREATE TRIGGER update_e2_nodes_updated_at BEFORE UPDATE ON e2.e2_nodes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_e2_subscriptions_updated_at BEFORE UPDATE ON e2.e2_subscriptions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_a1_policies_updated_at BEFORE UPDATE ON a1.a1_policies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_a1_enrichment_info_updated_at BEFORE UPDATE ON a1.a1_enrichment_info FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_xapps_updated_at BEFORE UPDATE ON xapp.xapps FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_xapp_instances_updated_at BEFORE UPDATE ON xapp.xapp_instances FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
CREATE OR REPLACE VIEW monitoring.system_overview AS
SELECT 
    'e2_nodes' as component,
    COUNT(*) as total_count,
    COUNT(CASE WHEN connection_status = 'connected' THEN 1 END) as healthy_count
FROM e2.e2_nodes
UNION ALL
SELECT 
    'a1_policies' as component,
    COUNT(*) as total_count,
    COUNT(CASE WHEN enforcement_status = 'enforced' THEN 1 END) as healthy_count
FROM a1.a1_policies
UNION ALL
SELECT 
    'xapp_instances' as component,
    COUNT(*) as total_count,
    COUNT(CASE WHEN status = 'running' THEN 1 END) as healthy_count
FROM xapp.xapp_instances
UNION ALL
SELECT 
    'active_alarms' as component,
    COUNT(*) as total_count,
    COUNT(CASE WHEN severity = 'critical' THEN 1 END) as healthy_count
FROM o1.o1_alarms 
WHERE status = 'active';

-- Grant permissions on views
GRANT SELECT ON monitoring.system_overview TO ric_user;

-- Analyze tables for better query planning
ANALYZE;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Database initialization completed successfully!';
    RAISE NOTICE 'Created schemas: e2, a1, o1, xapp, monitoring';
    RAISE NOTICE 'Created % tables with proper indexes and triggers', 
                 (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema IN ('e2', 'a1', 'o1', 'xapp', 'monitoring'));
END $$;