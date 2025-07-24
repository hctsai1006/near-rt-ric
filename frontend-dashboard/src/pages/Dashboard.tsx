import React from 'react'
import { motion } from 'framer-motion'
import { useQuery } from '@tanstack/react-query'
import { 
  Activity, 
  Radio, 
  Server, 
  Zap, 
  AlertTriangle, 
  CheckCircle, 
  TrendingUp,
  Network,
  Database,
  Cpu
} from 'lucide-react'
import StatsCard from '../components/StatsCard.tsx'
import NetworkTopology from '../components/NetworkTopology.tsx'
import RealTimeMetrics from '../components/RealTimeMetrics.tsx'
import AlarmsSummary from '../components/AlarmsSummary.tsx'
import { api } from '../services/api'
import { useRealTimeMetrics } from '../hooks/useRealTimeMetrics.ts'

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    },
  },
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
}

export default function Dashboard() {
  // Fetch overall system status
  const { data: systemStatus, isLoading: isLoadingStatus } = useQuery({
    queryKey: ['system-status'],
    queryFn: api.getSystemStatus,
  })

  // Fetch network functions overview
  const { data: networkFunctions, isLoading: isLoadingNFs } = useQuery({
    queryKey: ['network-functions', 'overview'],
    queryFn: api.getNetworkFunctionsOverview,
  })

  // Real-time metrics from WebSocket
  const { metrics, isConnected } = useRealTimeMetrics()

  const stats = [
    {
      name: 'Near-RT RIC Status',
      value: systemStatus?.nearRtRic?.status || 'Unknown',
      icon: Radio,
      color: systemStatus?.nearRtRic?.status === 'Active' ? 'green' : 'red',
      change: '+2%',
      changeType: 'positive' as const,
    },
    {
      name: 'Active xApps',
      value: networkFunctions?.xapps?.active || 0,
      icon: Zap,
      color: 'blue',
      change: `+${networkFunctions?.xapps?.recentlyDeployed || 0}`,
      changeType: 'positive' as const,
    },
    {
      name: 'E2 Connections',
      value: networkFunctions?.e2Connections?.total || 0,
      icon: Network,
      color: 'purple',
      change: `${networkFunctions?.e2Connections?.healthy || 0}/${networkFunctions?.e2Connections?.total || 0} healthy`,
      changeType: networkFunctions?.e2Connections?.healthy === networkFunctions?.e2Connections?.total ? 'positive' : 'negative' as const,
    },
    {
      name: 'SMO Components',
      value: `${systemStatus?.smo?.activeComponents || 0}/${systemStatus?.smo?.totalComponents || 0}`,
      icon: Server,
      color: 'orange',
      change: systemStatus?.smo?.status || 'Unknown',
      changeType: systemStatus?.smo?.status === 'Operational' ? 'positive' : 'negative' as const,
    },
    {
      name: 'Active Alarms',
      value: systemStatus?.alarms?.critical + systemStatus?.alarms?.major || 0,
      icon: AlertTriangle,
      color: systemStatus?.alarms?.critical > 0 ? 'red' : systemStatus?.alarms?.major > 0 ? 'yellow' : 'green',
      change: `${systemStatus?.alarms?.critical || 0} critical`,
      changeType: systemStatus?.alarms?.critical > 0 ? 'negative' : 'neutral' as const,
    },
    {
      name: 'Throughput',
      value: `${metrics?.throughput?.value || 0} ${metrics?.throughput?.unit || 'Mbps'}`,
      icon: TrendingUp,
      color: 'green',
      change: `${metrics?.throughput?.change || 0}%`,
      changeType: (metrics?.throughput?.change || 0) >= 0 ? 'positive' : 'negative' as const,
    },
    {
      name: 'Latency',
      value: `${metrics?.latency?.average || 0}ms`,
      icon: Activity,
      color: metrics?.latency?.average < 10 ? 'green' : metrics?.latency?.average < 50 ? 'yellow' : 'red',
      change: `${metrics?.latency?.change || 0}%`,
      changeType: (metrics?.latency?.change || 0) <= 0 ? 'positive' : 'negative' as const,
    },
    {
      name: 'Resource Usage',
      value: `${metrics?.resources?.cpu || 0}%`,
      icon: Cpu,
      color: metrics?.resources?.cpu < 70 ? 'green' : metrics?.resources?.cpu < 85 ? 'yellow' : 'red',
      change: `${metrics?.resources?.memory || 0}% memory`,
      changeType: 'neutral' as const,
    },
  ]

  return (
    <motion.div
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Header */}
      <motion.div variants={itemVariants} className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">O-RAN Operations Dashboard</h1>
          <p className="mt-2 text-gray-600">
            Real-time monitoring and control of Near-RT RIC and O-RAN network functions
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <div className={`h-3 w-3 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
          <span className="text-sm text-gray-600">
            {isConnected ? 'Real-time connected' : 'Connection lost'}
          </span>
        </div>
      </motion.div>

      {/* Stats Grid */}
      <motion.div 
        variants={itemVariants}
        className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4"
      >
        {stats.map((stat, index) => (
          <StatsCard
            key={stat.name}
            {...stat}
            isLoading={isLoadingStatus || isLoadingNFs}
            delay={index * 0.1}
          />
        ))}
      </motion.div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        {/* Network Topology */}
        <motion.div variants={itemVariants} className="xl:col-span-2">
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">Network Topology</h2>
              <div className="flex items-center space-x-2">
                <div className="h-2 w-2 bg-green-500 rounded-full animate-pulse" />
                <span className="text-sm text-gray-500">Live</span>
              </div>
            </div>
            <NetworkTopology 
              networkFunctions={networkFunctions}
              isLoading={isLoadingNFs}
            />
          </div>
        </motion.div>

        {/* Alarms Summary */}
        <motion.div variants={itemVariants}>
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Active Alarms</h2>
            <AlarmsSummary 
              alarms={systemStatus?.alarms}
              isLoading={isLoadingStatus}
            />
          </div>
        </motion.div>

        {/* Real-time Metrics */}
        <motion.div variants={itemVariants} className="xl:col-span-3">
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">Real-time Performance Metrics</h2>
              <div className="flex items-center space-x-4 text-sm text-gray-500">
                <span>Last updated: {new Date().toLocaleTimeString()}</span>
                <button className="text-blue-600 hover:text-blue-800">
                  Configure
                </button>
              </div>
            </div>
            <RealTimeMetrics 
              metrics={metrics}
              isConnected={isConnected}
            />
          </div>
        </motion.div>
      </div>

      {/* System Health Indicators */}
      <motion.div variants={itemVariants}>
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">System Health</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex items-center space-x-3">
              <CheckCircle className="h-5 w-5 text-green-500" />
              <div>
                <p className="text-sm font-medium text-gray-900">E2 Interface</p>
                <p className="text-sm text-gray-500">All connections healthy</p>
              </div>
            </div>
            <div className="flex items-center space-x-3">
              <CheckCircle className="h-5 w-5 text-green-500" />
              <div>
                <p className="text-sm font-medium text-gray-900">A1 Interface</p>
                <p className="text-sm text-gray-500">Policy management active</p>
              </div>
            </div>
            <div className="flex items-center space-x-3">
              <CheckCircle className="h-5 w-5 text-green-500" />
              <div>
                <p className="text-sm font-medium text-gray-900">O1 Interface</p>
                <p className="text-sm text-gray-500">Management plane operational</p>
              </div>
            </div>
          </div>
        </div>
      </motion.div>
    </motion.div>
  )
}