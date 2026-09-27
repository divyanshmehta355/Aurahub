"use client";

import React from 'react';
import { Line } from 'react-chartjs-2';
import { useTheme } from "next-themes";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

const TimeSeriesChart = ({ timeSeries }) => {
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';
  const textColor = isDark ? '#ffffff' : '#000000';
  const gridColor = isDark ? '#333333' : '#e5e5e5';
  const primaryColor = isDark ? '#ffffff' : '#000000';
  const primaryBg = isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';

  if (!timeSeries || timeSeries.length === 0) return null;

  const labels = timeSeries.map(d => d.date);
  const viewsData = timeSeries.map(d => d.views);
  const likesData = timeSeries.map(d => d.likes);

  const data = {
    labels,
    datasets: [
      {
        label: 'Views',
        data: viewsData,
        borderColor: primaryColor,
        backgroundColor: primaryBg,
        borderWidth: 2,
        tension: 0.4,
        fill: true,
        pointBackgroundColor: primaryColor,
        pointRadius: 2,
        pointHoverRadius: 5,
      },
      {
        label: 'Likes',
        data: likesData,
        borderColor: isDark ? '#a1a1aa' : '#71717a', // Neutral secondary
        backgroundColor: isDark ? 'rgba(161, 161, 170, 0.1)' : 'rgba(113, 113, 122, 0.1)',
        borderWidth: 2,
        tension: 0.4,
        fill: true,
        pointBackgroundColor: isDark ? '#a1a1aa' : '#71717a',
        pointRadius: 2,
        pointHoverRadius: 5,
      },
    ],
  };

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false,
    },
    plugins: {
      legend: {
        position: 'top',
        labels: {
          color: textColor,
          usePointStyle: true,
          font: {
            family: "'Inter', sans-serif",
            weight: '500'
          }
        }
      },
      tooltip: {
        backgroundColor: isDark ? 'rgba(0, 0, 0, 0.9)' : 'rgba(255, 255, 255, 0.9)',
        titleColor: isDark ? '#ffffff' : '#000000',
        bodyColor: isDark ? '#e5e5e5' : '#333333',
        borderColor: gridColor,
        borderWidth: 1,
        padding: 12,
        boxPadding: 6,
        usePointStyle: true,
      }
    },
    scales: {
        y: {
            beginAtZero: true,
            grid: {
              color: gridColor,
              drawBorder: false,
            },
            ticks: {
              color: textColor,
              precision: 0,
            }
        },
        x: {
            grid: {
              display: false,
              drawBorder: false,
            },
            ticks: {
              color: textColor,
              maxTicksLimit: 10,
            }
        }
    }
  };

  return (
    <div className="w-full h-[400px]">
      <Line options={options} data={data} />
    </div>
  );
};

export default TimeSeriesChart;
