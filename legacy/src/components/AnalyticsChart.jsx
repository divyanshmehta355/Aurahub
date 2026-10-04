"use client";

import React from 'react';
import { Bar } from 'react-chartjs-2';
import { useTheme } from "next-themes";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js';

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
);

const AnalyticsChart = ({ videos }) => {
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';
  const textColor = isDark ? '#ffffff' : '#000000';
  const gridColor = isDark ? '#333333' : '#e5e5e5';
  const primaryColor = isDark ? 'rgba(255, 255, 255, 0.9)' : 'rgba(0, 0, 0, 0.9)';

  const topVideos = (videos || []).slice(0, 10);
  const labels = topVideos.map(video => video.title);
  const viewsData = topVideos.map(video => video.views);
  const likesData = topVideos.map(video => video.likesCount);
  const commentsData = topVideos.map(video => video.commentCount);

  const data = {
    labels,
    datasets: [
      {
        label: 'Views',
        data: viewsData,
        backgroundColor: primaryColor,
        borderColor: isDark ? '#ffffff' : '#000000',
        borderWidth: 1,
        borderRadius: 4,
      },
      {
        label: 'Likes',
        data: likesData,
        backgroundColor: isDark ? 'rgba(161, 161, 170, 0.7)' : 'rgba(113, 113, 122, 0.7)', // Neutral secondary
        borderColor: isDark ? '#a1a1aa' : '#71717a',
        borderWidth: 1,
        borderRadius: 4,
      },
      {
        label: 'Comments',
        data: commentsData,
        backgroundColor: isDark ? 'rgba(82, 82, 91, 0.7)' : 'rgba(161, 161, 170, 0.7)', // Neutral tertiary
        borderColor: isDark ? '#52525b' : '#a1a1aa',
        borderWidth: 1,
        borderRadius: 4,
      },
    ],
  };

  const options = {
    responsive: true,
    indexAxis: 'y',
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top',
        labels: {
          color: textColor,
          font: {
            family: "'Inter', sans-serif",
            weight: '500'
          }
        }
      },
      title: {
        display: true,
        text: 'Top 10 Videos by Lifetime Views',
        color: textColor,
        font: {
          family: "'Inter', sans-serif",
          size: 16,
          weight: 'bold'
        }
      },
    },
    scales: {
        x: {
            beginAtZero: true,
            grid: {
              color: gridColor,
            },
            ticks: {
              color: textColor,
            }
        },
        y: {
            grid: {
              display: false,
            },
            ticks: {
              color: textColor,
            }
        }
    }
  };

  return <div className="h-[400px] w-full"><Bar options={options} data={data} /></div>;
};

export default AnalyticsChart;