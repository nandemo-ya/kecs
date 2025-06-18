const express = require('express');
const { DynamoDBClient } = require('@aws-sdk/client-dynamodb');
const { DynamoDBDocumentClient, PutCommand, GetCommand, ScanCommand } = require('@aws-sdk/lib-dynamodb');
const { ServiceDiscoveryClient, RegisterInstanceCommand } = require('@aws-sdk/client-servicediscovery');
const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');
const { v4: uuidv4 } = require('uuid');

const app = express();
app.use(express.json());

// Configuration
const PORT = process.env.PORT || 3001;
const SERVICE_NAME = process.env.SERVICE_NAME || 'user-service';
const TABLE_NAME = process.env.DYNAMODB_TABLE || 'users';
const LOCALSTACK_ENDPOINT = process.env.LOCALSTACK_ENDPOINT || 'http://localhost:4566';
const JWT_SECRET = process.env.JWT_SECRET || 'test-secret-key';
const SERVICE_ID = process.env.SERVICE_ID || 'srv-users';
const INSTANCE_ID = process.env.INSTANCE_ID || uuidv4();

// DynamoDB client
const dynamoClient = new DynamoDBClient({
  endpoint: LOCALSTACK_ENDPOINT,
  region: 'us-east-1',
  credentials: {
    accessKeyId: 'test',
    secretAccessKey: 'test'
  }
});

const docClient = DynamoDBDocumentClient.from(dynamoClient);

// Service Discovery client
const serviceDiscovery = new ServiceDiscoveryClient({
  endpoint: LOCALSTACK_ENDPOINT,
  region: 'us-east-1',
  credentials: {
    accessKeyId: 'test',
    secretAccessKey: 'test'
  }
});

// Initialize DynamoDB table
async function initializeTable() {
  // In production, table would be created via CloudFormation/Terraform
  console.log(`Using DynamoDB table: ${TABLE_NAME}`);
}

// Register with service discovery
async function registerService() {
  try {
    const command = new RegisterInstanceCommand({
      ServiceId: SERVICE_ID,
      InstanceId: INSTANCE_ID,
      Attributes: {
        AWS_INSTANCE_IPV4: process.env.HOSTNAME || 'localhost',
        AWS_INSTANCE_PORT: PORT.toString(),
        VERSION: '1.0.0',
        HEALTH_CHECK_URL: `/health`
      }
    });
    
    await serviceDiscovery.send(command);
    console.log(`Registered with service discovery: ${SERVICE_NAME}`);
  } catch (error) {
    console.error('Failed to register with service discovery:', error);
  }
}

// Health check
app.get('/health', async (req, res) => {
  try {
    // Test DynamoDB connection
    await docClient.send(new ScanCommand({
      TableName: TABLE_NAME,
      Limit: 1
    }));
    
    res.json({ 
      status: 'healthy',
      service: SERVICE_NAME,
      instance: INSTANCE_ID,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(503).json({ 
      status: 'unhealthy',
      error: error.message 
    });
  }
});

// Create user
app.post('/users', async (req, res) => {
  const { email, password, name } = req.body;
  
  if (!email || !password || !name) {
    return res.status(400).json({ error: 'Missing required fields' });
  }
  
  try {
    const userId = uuidv4();
    const hashedPassword = await bcrypt.hash(password, 10);
    
    const user = {
      id: userId,
      email,
      name,
      password: hashedPassword,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
    
    await docClient.send(new PutCommand({
      TableName: TABLE_NAME,
      Item: user,
      ConditionExpression: 'attribute_not_exists(email)'
    }));
    
    // Generate JWT token
    const token = jwt.sign({ userId, email }, JWT_SECRET, { expiresIn: '24h' });
    
    // Remove password from response
    delete user.password;
    
    res.status(201).json({ user, token });
  } catch (error) {
    if (error.name === 'ConditionalCheckFailedException') {
      res.status(409).json({ error: 'User already exists' });
    } else {
      res.status(500).json({ error: error.message });
    }
  }
});

// Get user
app.get('/users/:id', async (req, res) => {
  const { id } = req.params;
  
  try {
    const result = await docClient.send(new GetCommand({
      TableName: TABLE_NAME,
      Key: { id }
    }));
    
    if (!result.Item) {
      return res.status(404).json({ error: 'User not found' });
    }
    
    // Remove password from response
    delete result.Item.password;
    
    res.json(result.Item);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Login
app.post('/login', async (req, res) => {
  const { email, password } = req.body;
  
  if (!email || !password) {
    return res.status(400).json({ error: 'Missing email or password' });
  }
  
  try {
    // Find user by email (in production, use GSI)
    const result = await docClient.send(new ScanCommand({
      TableName: TABLE_NAME,
      FilterExpression: 'email = :email',
      ExpressionAttributeValues: {
        ':email': email
      }
    }));
    
    if (result.Items.length === 0) {
      return res.status(401).json({ error: 'Invalid credentials' });
    }
    
    const user = result.Items[0];
    const validPassword = await bcrypt.compare(password, user.password);
    
    if (!validPassword) {
      return res.status(401).json({ error: 'Invalid credentials' });
    }
    
    // Generate JWT token
    const token = jwt.sign({ userId: user.id, email }, JWT_SECRET, { expiresIn: '24h' });
    
    // Remove password from response
    delete user.password;
    
    res.json({ user, token });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Validate token (internal endpoint for other services)
app.post('/validate', (req, res) => {
  const { token } = req.body;
  
  if (!token) {
    return res.status(400).json({ error: 'Missing token' });
  }
  
  try {
    const decoded = jwt.verify(token, JWT_SECRET);
    res.json({ valid: true, userId: decoded.userId, email: decoded.email });
  } catch (error) {
    res.json({ valid: false, error: error.message });
  }
});

// Start server
app.listen(PORT, async () => {
  await initializeTable();
  await registerService();
  console.log(`User Service listening on port ${PORT}`);
  console.log(`DynamoDB table: ${TABLE_NAME}`);
  console.log(`LocalStack endpoint: ${LOCALSTACK_ENDPOINT}`);
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('SIGTERM received, shutting down gracefully');
  // Deregister from service discovery
  process.exit(0);
});