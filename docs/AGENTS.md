# AGENTS Configuration for O-RAN Near RT RIC xApp Deployment Project

This repository contains a comprehensive O-RAN (Open Radio Access Network) Near Real-Time RAN Intelligent Controller (Near-RT RIC) platform designed for xApp deployment and management. This AGENTS.md file provides essential context for AI coding assistants to understand the project structure and work effectively within this telecommunications domain.

## Project Overview

**Project Name:** near-rt-ric  
**Domain:** O-RAN 5G/LTE Telecommunications  
**Purpose:** Near Real-Time RAN Intelligent Controller for xApp deployment  
**Architecture:** Cloud-native, Kubernetes-based platform  
**Response Time Requirements:** 10ms - 1s (Near Real-Time)  
**Languages:** Go (42.0%), TypeScript (29.1%), HTML (16.9%), CSS (6.8%), SCSS (3.2%), Shell (0.9%)

## Architecture Components

### 1. dashboard-master/dashboard-master
- **Technology Stack:** Kubernetes Dashboard (Modified)
- **Backend:** Go-based API server with Gemini integration
- **Frontend:** Angular-based web UI 
- **Purpose:** Management interface for Near-RT RIC platform
- **Key Components:**
  - Go backend server (`src/app/backend/dashboard.go`)
  - Angular frontend (`src/app/frontend/`)
  - Kubernetes integration APIs
  - RBAC authentication system
  - Metrics and monitoring interfaces

### 2. xAPP_dashboard-master  
- **Technology Stack:** Angular 13.3.x
- **Purpose:** Specialized dashboard for xApp lifecycle management
- **Components:**
  - Front-page component (landing interface)
  - Home component (main dashboard)
  - Tags component (xApp categorization)
  - Image history component (container management)
  - Service layer for API communication

## O-RAN Domain Knowledge

### Key Concepts
- **Near-RT RIC:** Near Real-Time RAN Intelligent Controller
- **xApp:** Extended Applications running on Near-RT RIC
- **E2 Interface:** Communication protocol between RIC and RAN nodes
- **A1 Interface:** Communication between Non-RT RIC and Near-RT RIC
- **O1 Interface:** Management interface for FCAPS operations

### Technical Specifications
- **Latency Requirements:** 10ms to 1 second response time
- **Scalability:** Support for hundreds to thousands of network sites
- **Standards Compliance:** O-RAN Alliance specifications
- **Container Platform:** Kubernetes orchestration
- **Communication Protocols:** E2AP, A1 Policy, O1 Management

## Development Guidelines

### Code Organization
```
├── dashboard-master/dashboard-master/    # Kubernetes Dashboard (Modified)
│   ├── src/app/backend/                 # Go backend services
│   ├── src/app/frontend/                # Angular frontend
│   ├── aio/                            # Deployment configurations
│   ├── docs/                           # Documentation
│   └── i18n/                           # Internationalization
├── xAPP_dashboard-master/              # xApp Management Dashboard
│   ├── src/app/                        # Angular application
│   │   ├── front-page/                 # Landing page
│   │   ├── home/                       # Main dashboard
│   │   ├── tags/                       # xApp categorization
│   │   └── imagehistory/               # Container lifecycle
│   └── Dockerfile                      # Container build
└── README.md                          # Project documentation
```

### Coding Standards
- **Go Code:** Follow Go best practices, use dependency injection
- **TypeScript/Angular:** Strict mode, reactive forms, proper typing
- **Docker:** Multi-stage builds, minimal base images
- **Kubernetes:** Helm charts, resource limits, health checks

### Testing Requirements
- **Unit Tests:** Minimum 70% code coverage
- **Integration Tests:** E2E testing for critical workflows
- **Performance Tests:** Latency validation for real-time requirements
- **Security Tests:** RBAC, container scanning, network policies

### Deployment Considerations
- **Environment Setup:** Kubernetes cluster with RBAC
- **Dependencies:** Node.js 16.14.2+, Go 1.17+, Angular CLI
- **Build Process:** Make-based build system
- **Containerization:** Docker with multi-architecture support

## AI Assistant Instructions

When working on this project, please:

### 1. Maintain O-RAN Compliance
- Ensure all changes align with O-RAN Alliance specifications
- Preserve E2, A1, and O1 interface compatibility
- Maintain near real-time performance characteristics

### 2. Follow Telecommunications Standards  
- Use appropriate telecommunications terminology
- Implement proper error handling for network failures
- Consider multi-vendor interoperability requirements

### 3. Code Quality Standards
- Maintain existing code architecture patterns
- Add comprehensive logging for troubleshooting
- Include proper documentation for complex algorithms
- Validate input parameters for telecommunications protocols

### 4. Security Considerations
- Implement proper RBAC controls
- Validate all external API inputs
- Use secure communication channels
- Follow container security best practices

### 5. Performance Requirements
- Optimize for low-latency operations (10ms-1s)
- Consider memory usage for embedded environments  
- Implement efficient data structures for real-time processing
- Cache frequently accessed telecommunications data

## Pull Request Guidelines

### Commit Message Format
```
<component>: <type>: <description>

[optional body explaining the change]

Component: dashboard|xapp-dashboard|deployment|docs
Type: feat|fix|perf|refactor|docs|test|ci
```

### Example Commit Messages
```
dashboard: feat: add E2 interface monitoring dashboard
xapp-dashboard: fix: resolve xApp lifecycle state synchronization  
deployment: perf: optimize Kubernetes resource allocation
docs: docs: update O-RAN architecture documentation
```

### PR Description Template
```markdown
## Changes Made
- Brief description of changes

## O-RAN Impact Assessment  
- Interface compatibility: [E2/A1/O1]
- Performance impact: [latency/throughput]
- Standards compliance: [O-RAN Alliance specs]

## Testing Performed
- [ ] Unit tests pass
- [ ] Integration tests with RAN simulators
- [ ] Performance validation (latency < 1s)
- [ ] Security scan completed

## Deployment Notes
- Configuration changes required: [Y/N]  
- Breaking changes: [Y/N]
- Migration steps: [if applicable]
```

## Environment Setup Scripts

### Prerequisites Installation
```bash
# Node.js and npm
curl -fsSL https://deb.nodesource.com/setup_16.x | sudo -E bash -
sudo apt-get install -y nodejs

# Go installation  
wget https://golang.org/dl/go1.17.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Angular CLI
npm install -g @angular/cli@13.3.3

# Docker and Kubernetes tools
sudo apt-get install docker.io kubectl helm
```

### Build Commands
```bash
# Dashboard build
cd dashboard-master/dashboard-master
make build

# xApp Dashboard build  
cd xAPP_dashboard-master
npm install
npm run build

# Run tests
make test        # Dashboard tests
npm run test     # Angular tests
```

## Troubleshooting Common Issues

### Build Issues
- Ensure Node.js version >= 16.14.2
- Clear npm cache: `npm cache clean --force`
- Update dependencies: `npm update`

### Runtime Issues  
- Check Kubernetes cluster status
- Verify RBAC permissions
- Monitor resource utilization
- Check network connectivity to RAN nodes

### Performance Issues
- Profile CPU and memory usage
- Optimize database queries
- Check network latency to E2 nodes
- Validate real-time scheduling

---

This configuration ensures that AI assistants understand the complex O-RAN telecommunications domain while maintaining code quality and operational requirements for mission-critical network infrastructure.