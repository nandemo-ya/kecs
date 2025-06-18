const express = require('express');
const axios = require('axios');
const rateLimit = require('express-rate-limit');
const { ServiceDiscoveryClient, DiscoverInstancesCommand } = require('@aws-sdk/client-servicediscovery');

const app = express();
app.use(express.json());

// Configuration
const PORT = process.env.PORT || 8080;
const NAMESPACE_ID = process.env.SERVICE_DISCOVERY_NAMESPACE || 'microservices';
const LOCALSTACK_ENDPOINT = process.env.LOCALSTACK_ENDPOINT || 'http://localhost:4566';

// Service Discovery client
const serviceDiscovery = new ServiceDiscoveryClient({
  endpoint: LOCALSTACK_ENDPOINT,
  region: 'us-east-1',
  credentials: {
    accessKeyId: 'test',
    secretAccessKey: 'test'
  }
});

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100 // limit each IP to 100 requests per windowMs
});

app.use('/api/', limiter);

// Service discovery cache
const serviceCache = new Map();
const CACHE_TTL = 60000; // 1 minute

async function discoverService(serviceName) {
  const cacheKey = serviceName;
  const cached = serviceCache.get(cacheKey);
  
  if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
    return cached.instances;
  }

  try {
    const command = new DiscoverInstancesCommand({
      NamespaceName: NAMESPACE_ID,
      ServiceName: serviceName,
      MaxResults: 10
    });
    
    const response = await serviceDiscovery.send(command);
    const instances = response.Instances || [];
    
    serviceCache.set(cacheKey, {
      instances,
      timestamp: Date.now()
    });
    
    return instances;
  } catch (error) {
    console.error(`Failed to discover service ${serviceName}:`, error);
    return [];
  }
}

async function callService(serviceName, path, method = 'GET', data = null) {
  const instances = await discoverService(serviceName);
  
  if (instances.length === 0) {
    throw new Error(`No instances found for service: ${serviceName}`);
  }
  
  // Simple round-robin selection
  const instance = instances[Math.floor(Math.random() * instances.length)];
  const url = `http://${instance.Attributes.AWS_INSTANCE_IPV4}:${instance.Attributes.AWS_INSTANCE_PORT}${path}`;
  
  try {
    const response = await axios({
      method,
      url,
      data,
      timeout: 5000
    });
    return response.data;
  } catch (error) {
    console.error(`Failed to call ${serviceName}:`, error.message);
    throw error;
  }
}

// Health check
app.get('/health', async (req, res) => {
  const services = ['user-service', 'order-service', 'storage-service', 'notification-service'];
  const health = { status: 'healthy', services: {} };
  
  for (const service of services) {
    try {
      const instances = await discoverService(service);
      health.services[service] = {
        status: instances.length > 0 ? 'available' : 'unavailable',
        instances: instances.length
      };
    } catch (error) {
      health.services[service] = {
        status: 'error',
        error: error.message
      };
    }
  }
  
  res.json(health);
});

// User service routes
app.post('/api/users', async (req, res) => {
  try {
    const result = await callService('user-service', '/users', 'POST', req.body);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.get('/api/users/:id', async (req, res) => {
  try {
    const result = await callService('user-service', `/users/${req.params.id}`);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Order service routes
app.post('/api/orders', async (req, res) => {
  try {
    const result = await callService('order-service', '/orders', 'POST', req.body);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.get('/api/orders/:id', async (req, res) => {
  try {
    const result = await callService('order-service', `/orders/${req.params.id}`);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Storage service routes
app.post('/api/storage/upload', async (req, res) => {
  try {
    const result = await callService('storage-service', '/upload', 'POST', req.body);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.get('/api/storage/:key', async (req, res) => {
  try {
    const result = await callService('storage-service', `/files/${req.params.key}`);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Start server
app.listen(PORT, () => {
  console.log(`API Gateway listening on port ${PORT}`);
  console.log(`LocalStack endpoint: ${LOCALSTACK_ENDPOINT}`);
  console.log(`Service Discovery namespace: ${NAMESPACE_ID}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully');
  process.exit(0);
});