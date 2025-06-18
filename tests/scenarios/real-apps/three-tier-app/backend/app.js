const express = require('express');
const { Pool } = require('pg');
const redis = require('redis');
const AWS = require('aws-sdk');

const app = express();
app.use(express.json());

// Database configuration from environment
const pool = new Pool({
  host: process.env.DB_HOST || 'localhost',
  port: process.env.DB_PORT || 5432,
  database: process.env.DB_NAME || 'testapp',
  user: process.env.DB_USER || 'postgres',
  password: process.env.DB_PASSWORD || 'postgres',
});

// Redis configuration
const redisClient = redis.createClient({
  url: `redis://${process.env.REDIS_HOST || 'localhost'}:${process.env.REDIS_PORT || 6379}`
});

// S3 client for LocalStack
const s3 = new AWS.S3({
  endpoint: process.env.AWS_ENDPOINT_URL || 'http://localhost:4566',
  s3ForcePathStyle: true,
  region: process.env.AWS_DEFAULT_REGION || 'us-east-1',
});

// Connect to Redis
redisClient.on('error', (err) => console.log('Redis Client Error', err));
redisClient.connect().catch(console.error);

// Health check endpoint
app.get('/health', async (req, res) => {
  try {
    // Check database connection
    const dbResult = await pool.query('SELECT 1');
    
    // Check Redis connection
    await redisClient.ping();
    
    res.json({ 
      status: 'healthy',
      database: 'connected',
      cache: 'connected',
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(503).json({ 
      status: 'unhealthy', 
      error: error.message 
    });
  }
});

// API endpoints
app.get('/api/users', async (req, res) => {
  try {
    // Check cache first
    const cachedUsers = await redisClient.get('users');
    if (cachedUsers) {
      return res.json({ source: 'cache', data: JSON.parse(cachedUsers) });
    }

    // Query database
    const result = await pool.query('SELECT * FROM users ORDER BY id');
    
    // Cache the result
    await redisClient.setEx('users', 60, JSON.stringify(result.rows));
    
    res.json({ source: 'database', data: result.rows });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.post('/api/users', async (req, res) => {
  const { name, email } = req.body;
  try {
    const result = await pool.query(
      'INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *',
      [name, email]
    );
    
    // Invalidate cache
    await redisClient.del('users');
    
    res.status(201).json(result.rows[0]);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// S3 integration endpoint
app.get('/api/files', async (req, res) => {
  try {
    const params = {
      Bucket: process.env.S3_BUCKET || 'test-bucket',
    };
    
    const data = await s3.listObjectsV2(params).promise();
    res.json({ files: data.Contents || [] });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Service discovery test endpoint
app.get('/api/services', async (req, res) => {
  const services = {
    backend: {
      host: process.env.HOSTNAME || 'unknown',
      port: process.env.PORT || 3000,
    },
    database: {
      host: process.env.DB_HOST,
      port: process.env.DB_PORT,
    },
    cache: {
      host: process.env.REDIS_HOST,
      port: process.env.REDIS_PORT,
    },
  };
  
  res.json({ services });
});

// Initialize database
async function initDB() {
  try {
    await pool.query(`
      CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
      )
    `);
    console.log('Database initialized');
  } catch (error) {
    console.error('Failed to initialize database:', error);
  }
}

const PORT = process.env.PORT || 3000;

app.listen(PORT, async () => {
  await initDB();
  console.log(`Backend API listening on port ${PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('SIGTERM received, shutting down gracefully');
  await pool.end();
  await redisClient.quit();
  process.exit(0);
});