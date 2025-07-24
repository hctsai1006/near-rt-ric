// Placeholder for StatsCard component
import React from 'react';

const StatsCard: React.FC<{ name: string, value: any, icon: any, color: string, change: string, changeType: 'positive' | 'negative' | 'neutral', isLoading: boolean, delay: number }> = ({ name, value }) => {
  return <div>{name}: {value}</div>;
};

export default StatsCard;
