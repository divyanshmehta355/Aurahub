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

  const labels = videos.map(video => video.title);
  const viewsData = videos.map(video => video.views);
  const likesData = videos.map(video => video.likesCount);
  const commentsData = videos.map(video => video.commentCount);

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
        text: 'Video Performance Overview',
        color: textColor,
        font: {
          family: "'Inter', sans-serif",
          size: 16,
          weight: 'bold'
        }
      },
    },
    scales: {
        y: {
            beginAtZero: true,
            grid: {
              color: gridColor,
            },
            ticks: {
              color: textColor,
            }
        },
        x: {
            grid: {
              display: false,
            },
            ticks: {
              color: textColor,
            }
        }
    }
  };

  return <Bar options={options} data={data} />;
};

export default AnalyticsChart;