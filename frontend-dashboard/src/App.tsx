import React, { Suspense } from 'react'
import { Routes, Route } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useNetworkFunctionDiscovery } from './hooks/useNetworkFunctionDiscovery'
import { useWebSocketConnection } from './hooks/useWebSocketConnection.ts'
import Layout from './components/Layout.tsx'
import LoadingSpinner from './components/LoadingSpinner.tsx'

// Lazy load pages for better performance
const Dashboard = React.lazy(() => import('./pages/Dashboard.tsx'))
const NetworkFunctions = React.lazy(() => import('./pages/NetworkFunctions.tsx'))
const XApps = React.lazy(() => import('./pages/XApps.tsx'))
const SMO = React.lazy(() => import('./pages/SMO.tsx'))
const Monitoring = React.lazy(() => import('./pages/Monitoring.tsx'))
const Alarms = React.lazy(() => import('./pages/Alarms.tsx'))
const Configuration = React.lazy(() => import('./pages/Configuration.tsx'))
const NotFound = React.lazy(() => import('./pages/NotFound.tsx'))

// Page transition animations
const pageVariants = {
  initial: { opacity: 0, y: 20 },
  in: { opacity: 1, y: 0 },
  out: { opacity: 0, y: -20 },
}

const pageTransition = {
  type: 'tween',
  ease: 'anticipate',
  duration: 0.3,
}

function App() {
  // Initialize real-time connections and auto-discovery
  const { networkFunctions, isLoading: isDiscovering } = useNetworkFunctionDiscovery()
  const { isConnected, connectionStatus } = useWebSocketConnection()

  return (
    <Layout 
      networkFunctions={networkFunctions}
      connectionStatus={{ isConnected, status: connectionStatus }}
      isDiscovering={isDiscovering}
    >
      <motion.div
        initial="initial"
        animate="in"
        exit="out"
        variants={pageVariants}
        transition={pageTransition}
        className="flex-1"
      >
        <Suspense fallback={<LoadingSpinner />}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/network-functions" element={<NetworkFunctions />} />
            <Route path="/xapps" element={<XApps />} />
            <Route path="/smo" element={<SMO />} />
            <Route path="/monitoring" element={<Monitoring />} />
            <Route path="/alarms" element={<Alarms />} />
            <Route path="/configuration" element={<Configuration />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </Suspense>
      </motion.div>
    </Layout>
  )
}

export default App