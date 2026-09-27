import React from 'react';
import { motion } from 'framer-motion';
import { FaEye, FaThumbsUp, FaComment, FaUsers } from 'react-icons/fa';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

const StatCard = ({ title, value, icon, index }) => (
  <motion.div
    initial={{ opacity: 0, y: 20 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.4, delay: index * 0.1 }}
  >
    <Card className="rounded-2xl shadow-xl transition-colors h-full flex flex-col justify-between">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        <div className="p-3 rounded-xl bg-muted text-foreground shadow-inner">
          {icon}
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-bold text-foreground">
          {value?.toLocaleString() || 0}
        </div>
      </CardContent>
    </Card>
  </motion.div>
);

const CreatorStatCards = ({ stats }) => {
  if (!stats) return null;

  const statItems = [
    { title: 'Total Views', value: stats.views, icon: <FaEye size={20} /> },
    { title: 'Total Likes', value: stats.likes, icon: <FaThumbsUp size={20} /> },
    { title: 'Comments', value: stats.comments, icon: <FaComment size={20} /> },
    { title: 'Subscribers', value: stats.subscribers, icon: <FaUsers size={20} /> },
  ];

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      {statItems.map((item, idx) => (
        <StatCard key={idx} index={idx} title={item.title} value={item.value} icon={item.icon} />
      ))}
    </div>
  );
};

export default CreatorStatCards;
