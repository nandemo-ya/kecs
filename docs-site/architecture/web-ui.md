# Web UI Architecture

## Overview

The KECS Web UI is a modern, responsive React application that provides a comprehensive interface for managing ECS resources. It's built with TypeScript for type safety and uses WebSocket connections for real-time updates.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Web UI Architecture                      │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Frontend (React/TypeScript)             │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │  Components  │  │    Hooks     │  │   Context    │  │  │
│  │  │              │  │              │  │              │  │  │
│  │  │  - Dashboard │  │  - useAPI    │  │  - Auth      │  │  │
│  │  │  - Clusters  │  │  - useWS     │  │  - Theme     │  │  │
│  │  │  - Services  │  │  - useQuery  │  │  - Settings  │  │  │
│  │  │  - Tasks     │  │  - useMetrics│  │  - WebSocket │  │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │  │
│  └─────────┼──────────────────┼──────────────────┼──────────┘  │
│            │                  │                  │              │
│  ┌─────────▼──────────────────▼──────────────────▼──────────┐  │
│  │                      State Management                      │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │    Zustand   │  │ React Query  │  │   WebSocket  │  │  │
│  │  │    Store     │  │    Cache     │  │    Store     │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                     API Client Layer                       │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │   REST API   │  │  WebSocket   │  │   GraphQL    │  │  │
│  │  │   Client     │  │   Client     │  │   Client     │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Build & Bundle System                   │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │     Vite     │  │  TypeScript  │  │   PostCSS    │  │  │
│  │  │              │  │   Compiler   │  │              │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Stack

### Core Technologies

- **React 19**: Latest React with concurrent features
- **TypeScript 5**: Type safety and better DX
- **Vite**: Fast build tool and dev server
- **React Router**: Client-side routing
- **Tailwind CSS**: Utility-first CSS framework
- **React Query**: Server state management
- **Zustand**: Client state management
- **WebSocket**: Real-time communication

### UI Libraries

- **Radix UI**: Unstyled, accessible components
- **Lucide React**: Icon library
- **React Hook Form**: Form management
- **Zod**: Schema validation
- **Recharts**: Data visualization
- **Monaco Editor**: Code editing

## Component Architecture

### Directory Structure

```
web-ui/
├── src/
│   ├── components/           # UI Components
│   │   ├── common/          # Shared components
│   │   ├── dashboard/       # Dashboard views
│   │   ├── clusters/        # Cluster management
│   │   ├── services/        # Service management
│   │   ├── tasks/           # Task management
│   │   └── task-definitions/ # Task definition management
│   ├── hooks/               # Custom React hooks
│   ├── contexts/            # React contexts
│   ├── stores/              # Zustand stores
│   ├── api/                 # API client code
│   ├── types/               # TypeScript types
│   ├── utils/               # Utility functions
│   ├── styles/              # Global styles
│   └── App.tsx              # Root component
├── public/                  # Static assets
├── tests/                   # Test files
└── vite.config.ts          # Vite configuration
```

### Component Patterns

#### Container/Presenter Pattern

```typescript
// Container Component
export function ClustersContainer() {
  const { data: clusters, isLoading, error } = useClusters();
  const createCluster = useCreateCluster();
  
  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorDisplay error={error} />;
  
  return (
    <ClustersView
      clusters={clusters}
      onCreateCluster={createCluster}
    />
  );
}

// Presenter Component
interface ClustersViewProps {
  clusters: Cluster[];
  onCreateCluster: (data: CreateClusterInput) => void;
}

export function ClustersView({ clusters, onCreateCluster }: ClustersViewProps) {
  return (
    <div className="space-y-4">
      <ClusterList clusters={clusters} />
      <CreateClusterButton onClick={onCreateCluster} />
    </div>
  );
}
```

#### Compound Components

```typescript
export const ServiceCard = Object.assign(ServiceCardRoot, {
  Header: ServiceCardHeader,
  Body: ServiceCardBody,
  Actions: ServiceCardActions,
  Metrics: ServiceCardMetrics,
});

// Usage
<ServiceCard service={service}>
  <ServiceCard.Header />
  <ServiceCard.Body>
    <ServiceCard.Metrics />
  </ServiceCard.Body>
  <ServiceCard.Actions />
</ServiceCard>
```

## State Management

### Zustand Store Architecture

```typescript
interface AppStore {
  // Auth state
  user: User | null;
  isAuthenticated: boolean;
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  
  // UI state
  theme: 'light' | 'dark' | 'system';
  sidebarOpen: boolean;
  toggleSidebar: () => void;
  
  // Settings
  settings: AppSettings;
  updateSettings: (settings: Partial<AppSettings>) => void;
}

export const useAppStore = create<AppStore>((set, get) => ({
  user: null,
  isAuthenticated: false,
  
  login: async (credentials) => {
    const user = await authAPI.login(credentials);
    set({ user, isAuthenticated: true });
  },
  
  logout: () => {
    authAPI.logout();
    set({ user: null, isAuthenticated: false });
  },
  
  theme: 'system',
  sidebarOpen: true,
  
  toggleSidebar: () => set((state) => ({ 
    sidebarOpen: !state.sidebarOpen 
  })),
  
  settings: defaultSettings,
  updateSettings: (newSettings) => set((state) => ({
    settings: { ...state.settings, ...newSettings }
  })),
}));
```

### React Query Configuration

```typescript
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      cacheTime: 10 * 60 * 1000, // 10 minutes
      retry: 3,
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 1,
    },
  },
});

// Query hooks
export function useClusters() {
  return useQuery({
    queryKey: ['clusters'],
    queryFn: clusterAPI.listClusters,
    refetchInterval: 30000, // 30 seconds
  });
}

export function useCreateCluster() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: clusterAPI.createCluster,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['clusters'] });
      toast.success(`Cluster ${data.clusterName} created`);
    },
    onError: (error) => {
      toast.error(`Failed to create cluster: ${error.message}`);
    },
  });
}
```

## API Integration

### REST API Client

```typescript
class APIClient {
  private baseURL: string;
  private headers: Record<string, string>;
  
  constructor(baseURL: string) {
    this.baseURL = baseURL;
    this.headers = {
      'Content-Type': 'application/x-amz-json-1.1',
    };
  }
  
  private async request<T>(
    action: string,
    method: string = 'POST',
    data?: any
  ): Promise<T> {
    const response = await fetch(`${this.baseURL}/v1/${action}`, {
      method,
      headers: {
        ...this.headers,
        'X-Amz-Target': `AmazonEC2ContainerServiceV20141113.${action}`,
      },
      body: data ? JSON.stringify(data) : undefined,
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new APIError(error.__type, error.message);
    }
    
    return response.json();
  }
  
  // Cluster operations
  async listClusters(params?: ListClustersInput): Promise<ListClustersOutput> {
    return this.request('ListClusters', 'POST', params);
  }
  
  async createCluster(params: CreateClusterInput): Promise<CreateClusterOutput> {
    return this.request('CreateCluster', 'POST', params);
  }
  
  // ... other operations
}

export const apiClient = new APIClient(import.meta.env.VITE_API_URL);
```

### WebSocket Client

```typescript
interface WebSocketMessage {
  id: string;
  type: string;
  timestamp: string;
  payload: any;
}

class WebSocketClient extends EventEmitter {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private subscriptions = new Map<string, Subscription>();
  
  connect(url: string, token?: string) {
    const wsUrl = token ? `${url}?token=${token}` : url;
    this.ws = new WebSocket(wsUrl);
    
    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit('connected');
      this.resubscribe();
    };
    
    this.ws.onmessage = (event) => {
      const message: WebSocketMessage = JSON.parse(event.data);
      this.handleMessage(message);
    };
    
    this.ws.onerror = (error) => {
      this.emit('error', error);
    };
    
    this.ws.onclose = () => {
      this.emit('disconnected');
      this.reconnect();
    };
  }
  
  subscribe(eventTypes: string[], filters?: any): string {
    const subscriptionId = generateId();
    const subscription = { eventTypes, filters };
    
    this.subscriptions.set(subscriptionId, subscription);
    
    if (this.isConnected()) {
      this.send({
        id: generateId(),
        type: 'subscribe',
        action: 'events',
        payload: { eventTypes, filters },
      });
    }
    
    return subscriptionId;
  }
  
  unsubscribe(subscriptionId: string) {
    this.subscriptions.delete(subscriptionId);
    
    if (this.isConnected()) {
      this.send({
        id: generateId(),
        type: 'unsubscribe',
        action: 'events',
        payload: { subscriptionId },
      });
    }
  }
  
  private handleMessage(message: WebSocketMessage) {
    switch (message.type) {
      case 'event':
        this.emit('event', message.payload);
        break;
      case 'error':
        this.emit('error', message.payload);
        break;
      case 'ping':
        this.send({ type: 'pong' });
        break;
    }
  }
  
  private reconnect() {
    if (this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      this.emit('max_reconnect_exceeded');
      return;
    }
    
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;
    
    setTimeout(() => {
      this.connect(this.url, this.token);
    }, delay);
  }
}

export const wsClient = new WebSocketClient();
```

### WebSocket React Hook

```typescript
export function useWebSocket(eventTypes: string[], filters?: any) {
  const [events, setEvents] = useState<Event[]>([]);
  const [connected, setConnected] = useState(false);
  
  useEffect(() => {
    const handleEvent = (event: Event) => {
      setEvents((prev) => [...prev, event]);
    };
    
    const handleConnected = () => setConnected(true);
    const handleDisconnected = () => setConnected(false);
    
    wsClient.on('event', handleEvent);
    wsClient.on('connected', handleConnected);
    wsClient.on('disconnected', handleDisconnected);
    
    const subscriptionId = wsClient.subscribe(eventTypes, filters);
    
    return () => {
      wsClient.off('event', handleEvent);
      wsClient.off('connected', handleConnected);
      wsClient.off('disconnected', handleDisconnected);
      wsClient.unsubscribe(subscriptionId);
    };
  }, [eventTypes, filters]);
  
  return { events, connected };
}

// Usage in component
function ServiceDetails({ serviceArn }: { serviceArn: string }) {
  const { events } = useWebSocket(['service', 'task'], {
    services: [serviceArn],
  });
  
  // Process events for this service
  const serviceEvents = events.filter(
    (e) => e.payload.service?.serviceArn === serviceArn
  );
  
  return (
    <div>
      <EventTimeline events={serviceEvents} />
    </div>
  );
}
```

## Routing Architecture

```typescript
const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    errorElement: <ErrorBoundary />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
      {
        path: 'clusters',
        element: <ClustersLayout />,
        children: [
          {
            index: true,
            element: <ClusterList />,
          },
          {
            path: ':clusterName',
            element: <ClusterDetails />,
            loader: clusterLoader,
          },
        ],
      },
      {
        path: 'services',
        element: <ServicesLayout />,
        children: [
          {
            index: true,
            element: <ServiceList />,
          },
          {
            path: ':clusterName/:serviceName',
            element: <ServiceDetails />,
            loader: serviceLoader,
          },
        ],
      },
      {
        path: 'tasks',
        element: <TasksLayout />,
        children: [
          {
            index: true,
            element: <TaskList />,
          },
          {
            path: ':taskArn',
            element: <TaskDetails />,
            loader: taskLoader,
          },
        ],
      },
      {
        path: 'task-definitions',
        element: <TaskDefinitionsLayout />,
        children: [
          {
            index: true,
            element: <TaskDefinitionList />,
          },
          {
            path: 'new',
            element: <CreateTaskDefinition />,
          },
          {
            path: ':family/:revision?',
            element: <TaskDefinitionDetails />,
            loader: taskDefinitionLoader,
          },
        ],
      },
    ],
  },
]);

// Loaders for data fetching
async function clusterLoader({ params }: LoaderFunctionArgs) {
  const cluster = await apiClient.describeClusters({
    clusters: [params.clusterName!],
  });
  
  if (!cluster.clusters?.[0]) {
    throw new Response('Cluster not found', { status: 404 });
  }
  
  return cluster.clusters[0];
}
```

## Performance Optimization

### Code Splitting

```typescript
// Lazy load heavy components
const MonacoEditor = lazy(() => import('@monaco-editor/react'));
const MetricsChart = lazy(() => import('./components/MetricsChart'));

// Route-based code splitting
const routes = [
  {
    path: 'metrics',
    element: (
      <Suspense fallback={<LoadingSpinner />}>
        <MetricsChart />
      </Suspense>
    ),
  },
];
```

### Memoization

```typescript
// Memoize expensive computations
const ServiceMetrics = memo(({ service }: { service: Service }) => {
  const metrics = useMemo(() => {
    return calculateServiceMetrics(service);
  }, [service.runningCount, service.desiredCount, service.pendingCount]);
  
  return <MetricsDisplay metrics={metrics} />;
});

// Memoize callbacks
const ServiceActions = ({ service, onUpdate }: ServiceActionsProps) => {
  const handleScale = useCallback((count: number) => {
    onUpdate({ desiredCount: count });
  }, [onUpdate]);
  
  const handleDeploy = useCallback((taskDefinition: string) => {
    onUpdate({ taskDefinition });
  }, [onUpdate]);
  
  return (
    <div>
      <ScaleButton onClick={handleScale} />
      <DeployButton onClick={handleDeploy} />
    </div>
  );
};
```

### Virtual Scrolling

```typescript
import { FixedSizeList } from 'react-window';

function TaskList({ tasks }: { tasks: Task[] }) {
  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => (
    <div style={style}>
      <TaskRow task={tasks[index]} />
    </div>
  );
  
  return (
    <FixedSizeList
      height={600}
      itemCount={tasks.length}
      itemSize={80}
      width="100%"
    >
      {Row}
    </FixedSizeList>
  );
}
```

## Build System

### Vite Configuration

```typescript
// vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [
    react({
      babel: {
        plugins: [
          ['@babel/plugin-proposal-decorators', { legacy: true }],
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@components': path.resolve(__dirname, './src/components'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@api': path.resolve(__dirname, './src/api'),
    },
  },
  build: {
    target: 'es2020',
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'ui-vendor': ['@radix-ui/react-dialog', '@radix-ui/react-dropdown-menu'],
          'chart-vendor': ['recharts'],
          'editor-vendor': ['@monaco-editor/react'],
        },
      },
    },
  },
  server: {
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
});
```

### Production Build

```bash
# Build script
#!/bin/bash
cd web-ui

# Install dependencies
npm ci

# Type check
npm run type-check

# Run tests
npm run test

# Build for production
npm run build

# Embed into Go binary
cd ../controlplane
go generate ./...
go build -tags webui
```

## Testing Strategy

### Unit Tests

```typescript
// Component test
describe('ClusterCard', () => {
  it('displays cluster information', () => {
    const cluster: Cluster = {
      clusterName: 'production',
      status: 'ACTIVE',
      runningTasksCount: 10,
      activeServicesCount: 3,
    };
    
    render(<ClusterCard cluster={cluster} />);
    
    expect(screen.getByText('production')).toBeInTheDocument();
    expect(screen.getByText('ACTIVE')).toBeInTheDocument();
    expect(screen.getByText('10 tasks')).toBeInTheDocument();
    expect(screen.getByText('3 services')).toBeInTheDocument();
  });
});

// Hook test
describe('useCluster', () => {
  it('fetches cluster data', async () => {
    const { result } = renderHook(() => useCluster('production'));
    
    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
    
    expect(result.current.data?.clusterName).toBe('production');
  });
});
```

### Integration Tests

```typescript
// API integration test
describe('Cluster API', () => {
  it('creates and deletes cluster', async () => {
    const clusterName = `test-${Date.now()}`;
    
    // Create cluster
    const created = await apiClient.createCluster({ clusterName });
    expect(created.cluster?.clusterName).toBe(clusterName);
    
    // Verify cluster exists
    const clusters = await apiClient.listClusters();
    expect(clusters.clusterArns).toContain(created.cluster?.clusterArn);
    
    // Delete cluster
    await apiClient.deleteCluster({ cluster: clusterName });
    
    // Verify cluster deleted
    const afterDelete = await apiClient.listClusters();
    expect(afterDelete.clusterArns).not.toContain(created.cluster?.clusterArn);
  });
});
```

### E2E Tests

```typescript
// Playwright test
test('create service workflow', async ({ page }) => {
  await page.goto('/services');
  
  // Click create button
  await page.click('button:has-text("Create Service")');
  
  // Fill form
  await page.fill('input[name="serviceName"]', 'test-service');
  await page.selectOption('select[name="cluster"]', 'production');
  await page.fill('input[name="desiredCount"]', '3');
  
  // Submit
  await page.click('button[type="submit"]');
  
  // Verify redirect
  await expect(page).toHaveURL(/\/services\/production\/test-service/);
  
  // Verify service displayed
  await expect(page.locator('h1')).toContainText('test-service');
});
```

## Security Considerations

### Content Security Policy

```typescript
const cspHeader = {
  'Content-Security-Policy': [
    "default-src 'self'",
    "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: https:",
    "font-src 'self'",
    "connect-src 'self' ws://localhost:* wss://localhost:*",
  ].join('; '),
};
```

### Authentication Flow

```typescript
function App() {
  const { isAuthenticated, isLoading } = useAuth();
  
  if (isLoading) {
    return <LoadingScreen />;
  }
  
  if (!isAuthenticated) {
    return <LoginPage />;
  }
  
  return <AuthenticatedApp />;
}

function useAuth() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  
  useEffect(() => {
    const checkAuth = async () => {
      try {
        const token = localStorage.getItem('auth-token');
        if (!token) {
          setIsAuthenticated(false);
          return;
        }
        
        const user = await apiClient.validateToken(token);
        setIsAuthenticated(!!user);
      } catch {
        setIsAuthenticated(false);
      } finally {
        setIsLoading(false);
      }
    };
    
    checkAuth();
  }, []);
  
  return { isAuthenticated, isLoading };
}
```

## Future Enhancements

1. **Progressive Web App**
   - Offline support with service workers
   - Push notifications
   - App manifest

2. **Advanced Features**
   - Drag-and-drop task definition builder
   - Visual service topology
   - Cost estimation
   - Performance profiling

3. **Accessibility**
   - WCAG 2.1 AA compliance
   - Keyboard navigation
   - Screen reader support
   - High contrast themes

4. **Internationalization**
   - Multi-language support
   - RTL layout support
   - Locale-specific formatting