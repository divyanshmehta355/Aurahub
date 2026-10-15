<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import {
		BarChart3,
		TrendingUp,
		Users,
		Eye,
		ThumbsUp,
		MessageSquare,
		Video,
		ArrowUpRight,
		Play,
		RefreshCw,
		Calendar,
		Sparkles,
		Award,
		Layers,
		Clock,
		Flame,
		PieChart,
		ChevronRight,
		ExternalLink,
		PlusCircle
	} from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Button } from '#lib/components/ui/button';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { goto } from '$app/navigation';

	const queryClient = useQueryClient();

	let selectedDays = $state<7 | 30 | 90>(30);
	let activeChartMetric = $state<'views' | 'likes' | 'combined'>('views');

	// Hover state for interactive SVG chart
	let hoveredIndex = $state<number | null>(null);

	const analyticsQuery = createQuery(
		() => ({
			queryKey: ['analytics', selectedDays],
			queryFn: async () => {
				if (!userState.user && userState.isLoaded) {
					goto('/login');
					return null;
				}
				return await fetchApi(`/creator/analytics?days=${selectedDays}`);
			},
			enabled: !!userState.user
		}),
		() => queryClient
	);

	function formatNumber(num: number | undefined | null): string {
		if (num === undefined || num === null) return '0';
		if (num >= 1_000_000) return (num / 1_000_000).toFixed(1) + 'M';
		if (num >= 1_000) return (num / 1_000).toFixed(1) + 'K';
		return num.toLocaleString();
	}

	function formatDuration(seconds: number | undefined | null): string {
		if (!seconds || seconds <= 0) return '0:00';
		const mins = Math.floor(seconds / 60);
		const secs = Math.floor(seconds % 60);
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return '';
		const d = new Date(dateStr);
		if (isNaN(d.getTime())) return dateStr;
		return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	// SVG Chart Dimensions
	const svgWidth = 840;
	const svgHeight = 260;
	const padLeft = 46;
	const padRight = 24;
	const padTop = 24;
	const padBottom = 38;
	const usableWidth = svgWidth - padLeft - padRight;
	const usableHeight = svgHeight - padTop - padBottom;

	// Computed Chart Data
	let timeSeriesData = $derived(analyticsQuery.data?.timeSeries || []);

	let maxChartVal = $derived.by(() => {
		if (timeSeriesData.length === 0) return 10;
		let max = 0;
		for (const pt of timeSeriesData) {
			const val =
				activeChartMetric === 'views'
					? pt.viewInteractions || 0
					: activeChartMetric === 'likes'
						? pt.likeInteractions || 0
						: (pt.viewInteractions || 0) + (pt.likeInteractions || 0);
			if (val > max) max = val;
		}
		return max > 0 ? Math.ceil(max * 1.2) : 10;
	});

	let chartPoints = $derived.by(() => {
		if (timeSeriesData.length === 0) return [];
		const n = timeSeriesData.length;
		return timeSeriesData.map((pt: any, i: number) => {
			const x = padLeft + (i / Math.max(1, n - 1)) * usableWidth;
			const val =
				activeChartMetric === 'views'
					? pt.viewInteractions || 0
					: activeChartMetric === 'likes'
						? pt.likeInteractions || 0
						: (pt.viewInteractions || 0) + (pt.likeInteractions || 0);
			const y = padTop + (1 - val / maxChartVal) * usableHeight;
			return { x, y, val, raw: pt, index: i };
		});
	});

	// Smooth cubic Bézier spline
	function buildSmoothPath(points: { x: number; y: number }[]): string {
		if (points.length === 0) return '';
		if (points.length === 1) return `M ${points[0].x} ${points[0].y}`;
		let d = `M ${points[0].x.toFixed(1)} ${points[0].y.toFixed(1)}`;
		for (let i = 0; i < points.length - 1; i++) {
			const p0 = points[i];
			const p1 = points[i + 1];
			const mx = ((p0.x + p1.x) / 2).toFixed(1);
			d += ` C ${mx} ${p0.y.toFixed(1)}, ${mx} ${p1.y.toFixed(1)}, ${p1.x.toFixed(1)} ${p1.y.toFixed(1)}`;
		}
		return d;
	}

	function buildAreaPath(points: { x: number; y: number }[]): string {
		if (points.length === 0) return '';
		const baselineY = (padTop + usableHeight).toFixed(1);
		const firstX = points[0].x.toFixed(1);
		const lastX = points[points.length - 1].x.toFixed(1);
		const linePath = buildSmoothPath(points);
		return `${linePath} L ${lastX} ${baselineY} L ${firstX} ${baselineY} Z`;
	}

	let chartPath = $derived(buildSmoothPath(chartPoints));
	let chartArea = $derived(buildAreaPath(chartPoints));

	let currentHoveredPoint = $derived(
		hoveredIndex !== null && chartPoints[hoveredIndex] ? chartPoints[hoveredIndex] : null
	);

	function handleChartMouseMove(e: MouseEvent) {
		const target = e.currentTarget as SVGSVGElement;
		const rect = target.getBoundingClientRect();
		const mouseX = e.clientX - rect.left;
		const normalizedX = (mouseX / rect.width) * svgWidth;

		if (chartPoints.length === 0) return;

		let closestIdx = 0;
		let minDiff = Infinity;
		for (let i = 0; i < chartPoints.length; i++) {
			const diff = Math.abs(chartPoints[i].x - normalizedX);
			if (diff < minDiff) {
				minDiff = diff;
				closestIdx = i;
			}
		}
		hoveredIndex = closestIdx;
	}

	function handleChartMouseLeave() {
		hoveredIndex = null;
	}

	// Category palette
	const categoryColors = [
		{ bg: 'bg-violet-500', text: 'text-violet-400', border: 'border-violet-500/30' },
		{ bg: 'bg-emerald-500', text: 'text-emerald-400', border: 'border-emerald-500/30' },
		{ bg: 'bg-rose-500', text: 'text-rose-400', border: 'border-rose-500/30' },
		{ bg: 'bg-amber-500', text: 'text-amber-400', border: 'border-amber-500/30' },
		{ bg: 'bg-sky-500', text: 'text-sky-400', border: 'border-sky-500/30' },
		{ bg: 'bg-indigo-500', text: 'text-indigo-400', border: 'border-indigo-500/30' }
	];
</script>

<svelte:head>
	<title>Channel Analytics - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-[1400px] space-y-8 px-4 py-6 sm:px-6 md:py-8">
	<!-- Top Bar / Header -->
	<div class="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
		<div class="flex items-center gap-4">
			{#if userState.user}
				<div class="relative">
					<img
						src={userState.user.avatar ||
							`https://api.dicebear.com/7.x/identicon/svg?seed=${userState.user.username}`}
						alt={userState.user.username}
						class="h-14 w-14 rounded-2xl border-2 border-primary/20 object-cover shadow-sm sm:h-16 sm:w-16"
					/>
					<div
						class="absolute -right-1 -bottom-1 flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm"
						title="Creator Channel"
					>
						<Sparkles class="h-3.5 w-3.5" />
					</div>
				</div>
			{/if}
			<div>
				<div class="flex items-center gap-2.5">
					<h1 class="text-2xl font-black tracking-tight sm:text-3xl">Channel Analytics</h1>
					<span
						class="rounded-full border border-primary/30 bg-primary/10 px-2.5 py-0.5 text-xs font-bold text-primary"
					>
						Studio
					</span>
				</div>
				<p class="text-muted-foreground mt-1 text-sm">
					Detailed audience growth, interaction trends, and content performance insights.
				</p>
			</div>
		</div>

		<!-- Action Controls & Time Period Selector -->
		<div class="flex flex-wrap items-center gap-3">
			<!-- Time Range Pills -->
			<div class="flex items-center rounded-2xl border border-border/80 bg-muted/40 p-1">
				{#each [7, 30, 90] as days}
					<button
						type="button"
						onclick={() => (selectedDays = days as 7 | 30 | 90)}
						class="rounded-xl px-3.5 py-1.5 text-xs font-bold transition-all {selectedDays === days
							? 'bg-card text-foreground shadow-sm'
							: 'text-muted-foreground hover:text-foreground'}"
					>
						{days} Days
					</button>
				{/each}
			</div>

			<Button
				variant="outline"
				size="sm"
				onclick={() => analyticsQuery.refetch()}
				class="h-9 gap-1.5 rounded-xl border-border/80 text-xs font-semibold"
				disabled={analyticsQuery.isFetching}
			>
				<RefreshCw class="h-3.5 w-3.5 {analyticsQuery.isFetching ? 'animate-spin' : ''}" />
				<span class="hidden sm:inline">Refresh</span>
			</Button>

			<Button
				href="/dashboard"
				variant="default"
				size="sm"
				class="h-9 gap-1.5 rounded-xl text-xs font-bold shadow-sm"
			>
				<Video class="h-3.5 w-3.5" />
				<span>Content Studio</span>
			</Button>
		</div>
	</div>

	<!-- Main Analytics Grid -->
	{#if analyticsQuery.isPending}
		<!-- Loading Skeletons -->
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each Array(4) as _}
				<Skeleton class="h-36 rounded-3xl" />
			{/each}
		</div>
		<Skeleton class="h-[380px] w-full rounded-3xl" />
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
			<Skeleton class="h-96 rounded-3xl lg:col-span-2" />
			<Skeleton class="h-96 rounded-3xl" />
		</div>
	{:else if analyticsQuery.isError}
		<!-- Error State -->
		<div
			class="flex flex-col items-center justify-center rounded-3xl border border-destructive/20 bg-destructive/5 p-12 text-center"
		>
			<div class="rounded-2xl bg-destructive/10 p-4 text-destructive">
				<BarChart3 class="h-10 w-10" />
			</div>
			<h2 class="mt-4 text-xl font-bold">Failed to load analytics</h2>
			<p class="text-muted-foreground mt-2 max-w-md text-sm">
				{analyticsQuery.error?.message ||
					'Unable to retrieve creator analytics. Please verify your connection.'}
			</p>
			<Button
				variant="outline"
				class="mt-6 rounded-2xl"
				onclick={() => analyticsQuery.refetch()}
			>
				Try Again
			</Button>
		</div>
	{:else if analyticsQuery.data}
		{@const data = analyticsQuery.data}
		{@const stats = data.lifetimeStats || {}}
		{@const period = data.periodActivity || {}}
		{@const totalLifetimeViews = stats.views || 0}

		{#if (stats.videos || 0) === 0}
			<!-- Empty State when user has 0 uploaded videos -->
			<div
				class="relative overflow-hidden rounded-3xl border border-border/80 bg-gradient-to-b from-card via-card to-muted/20 p-8 text-center shadow-sm sm:p-14"
			>
				<div class="mx-auto max-w-lg space-y-4">
					<div
						class="mx-auto flex h-20 w-20 items-center justify-center rounded-3xl bg-primary/10 text-primary shadow-inner"
					>
						<Video class="h-10 w-10" />
					</div>
					<h2 class="text-2xl font-black tracking-tight sm:text-3xl">Welcome to Your Studio!</h2>
					<p class="text-muted-foreground text-sm sm:text-base">
						You haven't uploaded any videos yet. Publish your first video to start tracking views,
						audience engagement, likes, and watch trends in real time.
					</p>
					<div class="pt-2">
						<Button
							href="/upload"
							size="lg"
							class="gap-2 rounded-2xl px-6 text-sm font-bold shadow-lg shadow-primary/20"
						>
							<PlusCircle class="h-4 w-4" />
							Upload First Video
						</Button>
					</div>
				</div>
			</div>
		{:else}
			<!-- 1. Executive Performance Cards (4 Ribbon KPI Cards) -->
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<!-- KPI 1: Views -->
				<div
					class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all duration-300 hover:border-violet-500/40 hover:shadow-lg hover:shadow-violet-500/5"
				>
					<div class="flex items-center justify-between">
						<span class="text-xs font-bold tracking-wider text-muted-foreground uppercase"
							>Channel Views</span
						>
						<div
							class="rounded-2xl bg-violet-500/10 p-2.5 text-violet-500 transition-transform group-hover:scale-110"
						>
							<Eye class="h-5 w-5" />
						</div>
					</div>
					<p class="mt-3 text-3xl font-black tracking-tight text-foreground">
						{formatNumber(stats.views)}
					</p>
					<div class="mt-2 flex items-center justify-between text-xs text-muted-foreground">
						<span class="flex items-center gap-1 font-semibold text-emerald-500">
							<TrendingUp class="h-3.5 w-3.5" />
							{formatNumber(period.viewInteractions)} in {selectedDays}d
						</span>
						<span>Avg {(stats.avgViewsPerVideo || 0).toFixed(0)}/vid</span>
					</div>
				</div>

				<!-- KPI 2: Subscribers -->
				<div
					class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all duration-300 hover:border-blue-500/40 hover:shadow-lg hover:shadow-blue-500/5"
				>
					<div class="flex items-center justify-between">
						<span class="text-xs font-bold tracking-wider text-muted-foreground uppercase"
							>Subscribers</span
						>
						<div
							class="rounded-2xl bg-blue-500/10 p-2.5 text-blue-500 transition-transform group-hover:scale-110"
						>
							<Users class="h-5 w-5" />
						</div>
					</div>
					<p class="mt-3 text-3xl font-black tracking-tight text-foreground">
						{formatNumber(stats.subscribers)}
					</p>
					<div class="mt-2 flex items-center justify-between text-xs text-muted-foreground">
						<span class="font-semibold text-foreground">{stats.videos} Videos</span>
						<span class="text-blue-400">Total Audience</span>
					</div>
				</div>

				<!-- KPI 3: Likes -->
				<div
					class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all duration-300 hover:border-rose-500/40 hover:shadow-lg hover:shadow-rose-500/5"
				>
					<div class="flex items-center justify-between">
						<span class="text-xs font-bold tracking-wider text-muted-foreground uppercase"
							>Likes Received</span
						>
						<div
							class="rounded-2xl bg-rose-500/10 p-2.5 text-rose-500 transition-transform group-hover:scale-110"
						>
							<ThumbsUp class="h-5 w-5" />
						</div>
					</div>
					<p class="mt-3 text-3xl font-black tracking-tight text-foreground">
						{formatNumber(stats.likes)}
					</p>
					<div class="mt-2 flex items-center justify-between text-xs text-muted-foreground">
						<span class="font-semibold text-rose-500">
							+{formatNumber(period.likeInteractions)} in {selectedDays}d
						</span>
						<span class="font-medium">
							{stats.views > 0 ? ((stats.likes / stats.views) * 100).toFixed(1) : '0'}% ratio
						</span>
					</div>
				</div>

				<!-- KPI 4: Engagement & Comments -->
				<div
					class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all duration-300 hover:border-amber-500/40 hover:shadow-lg hover:shadow-amber-500/5"
				>
					<div class="flex items-center justify-between">
						<span class="text-xs font-bold tracking-wider text-muted-foreground uppercase"
							>Community Feedback</span
						>
						<div
							class="rounded-2xl bg-amber-500/10 p-2.5 text-amber-500 transition-transform group-hover:scale-110"
						>
							<MessageSquare class="h-5 w-5" />
						</div>
					</div>
					<p class="mt-3 text-3xl font-black tracking-tight text-foreground">
						{formatNumber(stats.comments)}
					</p>
					<div class="mt-2 flex items-center justify-between text-xs text-muted-foreground">
						<span class="font-semibold text-amber-400">
							{(stats.overallEngagementRate || 0).toFixed(1)}% Eng. Rate
						</span>
						<span>Total Comments</span>
					</div>
				</div>
			</div>

			<!-- 2. Interactive Activity Trends Chart -->
			<div class="relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm">
				<!-- Chart Header & Mode Controls -->
				<div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
					<div>
						<div class="flex items-center gap-2">
							<h2 class="text-lg font-bold text-foreground">Interaction Trends</h2>
							<span class="rounded-md bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
								Last {selectedDays} Days
							</span>
						</div>
						<p class="text-muted-foreground mt-0.5 text-xs">
							Daily interactions and audience engagement over time.
						</p>
					</div>

					<!-- Metric Tabs for Chart -->
					<div class="flex items-center rounded-2xl border border-border/80 bg-muted/30 p-1">
						<button
							type="button"
							onclick={() => (activeChartMetric = 'views')}
							class="flex items-center gap-1.5 rounded-xl px-3 py-1 text-xs font-bold transition-all {activeChartMetric ===
							'views'
								? 'bg-violet-600 text-white shadow-sm'
								: 'text-muted-foreground hover:text-foreground'}"
						>
							<Eye class="h-3.5 w-3.5" />
							<span>Views</span>
						</button>
						<button
							type="button"
							onclick={() => (activeChartMetric = 'likes')}
							class="flex items-center gap-1.5 rounded-xl px-3 py-1 text-xs font-bold transition-all {activeChartMetric ===
							'likes'
								? 'bg-rose-600 text-white shadow-sm'
								: 'text-muted-foreground hover:text-foreground'}"
						>
							<ThumbsUp class="h-3.5 w-3.5" />
							<span>Likes</span>
						</button>
						<button
							type="button"
							onclick={() => (activeChartMetric = 'combined')}
							class="flex items-center gap-1.5 rounded-xl px-3 py-1 text-xs font-bold transition-all {activeChartMetric ===
							'combined'
								? 'bg-emerald-600 text-white shadow-sm'
								: 'text-muted-foreground hover:text-foreground'}"
						>
							<Layers class="h-3.5 w-3.5" />
							<span>Combined</span>
						</button>
					</div>
				</div>

				<!-- SVG Chart Display Container -->
				<div class="relative w-full">
					<svg
						viewBox="0 0 {svgWidth} {svgHeight}"
						class="h-64 w-full select-none overflow-visible sm:h-72"
						onmousemove={handleChartMouseMove}
						onmouseleave={handleChartMouseLeave}
						role="img"
						aria-label="Interactive Audience Activity Chart"
					>
						<defs>
							<!-- Violet Gradient (Views) -->
							<linearGradient id="gradient-views" x1="0" y1="0" x2="0" y2="1">
								<stop offset="0%" stop-color="#8b5cf6" stop-opacity="0.38" />
								<stop offset="100%" stop-color="#8b5cf6" stop-opacity="0.0" />
							</linearGradient>

							<!-- Rose Gradient (Likes) -->
							<linearGradient id="gradient-likes" x1="0" y1="0" x2="0" y2="1">
								<stop offset="0%" stop-color="#f43f5e" stop-opacity="0.38" />
								<stop offset="100%" stop-color="#f43f5e" stop-opacity="0.0" />
							</linearGradient>

							<!-- Emerald Gradient (Combined) -->
							<linearGradient id="gradient-combined" x1="0" y1="0" x2="0" y2="1">
								<stop offset="0%" stop-color="#10b981" stop-opacity="0.38" />
								<stop offset="100%" stop-color="#10b981" stop-opacity="0.0" />
							</linearGradient>
						</defs>

						<!-- Background Horizontal Guide Lines -->
						{#each [0, 0.25, 0.5, 0.75, 1] as fraction}
							{@const yPos = padTop + usableHeight * (1 - fraction)}
							{@const guideVal = Math.round(maxChartVal * fraction)}
							<line
								x1={padLeft}
								y1={yPos}
								x2={svgWidth - padRight}
								y2={yPos}
								stroke="currentColor"
								stroke-opacity="0.08"
								stroke-dasharray="3,3"
							/>
							<text
								x={padLeft - 8}
								y={yPos + 3.5}
								text-anchor="end"
								class="fill-muted-foreground text-[10px] font-semibold"
							>
								{guideVal}
							</text>
						{/each}

						<!-- Area Fill under curve -->
						{#if chartArea}
							<path
								d={chartArea}
								fill={activeChartMetric === 'views'
									? 'url(#gradient-views)'
									: activeChartMetric === 'likes'
										? 'url(#gradient-likes)'
										: 'url(#gradient-combined)'}
							/>
						{/if}

						<!-- Stroke Curve Line -->
						{#if chartPath}
							<path
								d={chartPath}
								fill="none"
								stroke={activeChartMetric === 'views'
									? '#8b5cf6'
									: activeChartMetric === 'likes'
										? '#f43f5e'
										: '#10b981'}
								stroke-width="3"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						{/if}

						<!-- Bottom X-axis Dates -->
						{#each chartPoints as pt, i}
							{#if i % Math.max(1, Math.floor(chartPoints.length / 7)) === 0 || i === chartPoints.length - 1}
								<text
									x={pt.x}
									y={svgHeight - 10}
									text-anchor="middle"
									class="fill-muted-foreground text-[10px] font-medium"
								>
									{pt.raw.date}
								</text>
							{/if}
						{/each}

						<!-- Interactive Hover Guide & Dots -->
						{#if currentHoveredPoint}
							<!-- Vertical guide line -->
							<line
								x1={currentHoveredPoint.x}
								y1={padTop}
								x2={currentHoveredPoint.x}
								y2={padTop + usableHeight}
								stroke="currentColor"
								stroke-opacity="0.3"
								stroke-dasharray="4,4"
								stroke-width="1.5"
							/>

							<!-- Outer Pulse Ring -->
							<circle
								cx={currentHoveredPoint.x}
								cy={currentHoveredPoint.y}
								r="8"
								fill={activeChartMetric === 'views'
									? '#8b5cf6'
									: activeChartMetric === 'likes'
										? '#f43f5e'
										: '#10b981'}
								fill-opacity="0.25"
								class="animate-ping"
							/>

							<!-- Main Center Dot -->
							<circle
								cx={currentHoveredPoint.x}
								cy={currentHoveredPoint.y}
								r="5"
								fill={activeChartMetric === 'views'
									? '#8b5cf6'
									: activeChartMetric === 'likes'
										? '#f43f5e'
										: '#10b981'}
								stroke="#ffffff"
								stroke-width="2.5"
							/>
						{/if}
					</svg>

					<!-- Floating Interactive Tooltip -->
					{#if currentHoveredPoint}
						{@const p = currentHoveredPoint.raw}
						{@const tooltipLeft = Math.min(
							Math.max(10, (currentHoveredPoint.x / svgWidth) * 100),
							88
						)}
						<div
							class="pointer-events-none absolute z-20 -top-2 flex -translate-x-1/2 flex-col rounded-2xl border border-border/80 bg-popover/95 p-3 text-popover-foreground shadow-xl backdrop-blur-md transition-all duration-75"
							style="left: {tooltipLeft}%;"
						>
							<div class="flex items-center justify-between gap-4 border-b border-border/60 pb-1.5">
								<span class="text-xs font-bold text-foreground">{p.date}</span>
								<span class="text-[10px] font-semibold text-muted-foreground">Day Summary</span>
							</div>
							<div class="mt-2 space-y-1 text-xs">
								<div class="flex items-center justify-between gap-3">
									<span class="flex items-center gap-1.5 text-violet-400 font-medium">
										<Eye class="h-3 w-3" /> Views:
									</span>
									<span class="font-bold text-foreground">{p.viewInteractions || 0}</span>
								</div>
								<div class="flex items-center justify-between gap-3">
									<span class="flex items-center gap-1.5 text-rose-400 font-medium">
										<ThumbsUp class="h-3 w-3" /> Likes:
									</span>
									<span class="font-bold text-foreground">{p.likeInteractions || 0}</span>
								</div>
							</div>
						</div>
					{/if}
				</div>

				<!-- Chart Footer Insights -->
				<div
					class="mt-4 flex flex-wrap items-center justify-between border-t border-border/60 pt-4 text-xs text-muted-foreground"
				>
					<div class="flex items-center gap-2">
						<Calendar class="h-3.5 w-3.5" />
						<span>Displaying {timeSeriesData.length} consecutive days of channel data</span>
					</div>
					<div class="flex items-center gap-4">
						<span class="flex items-center gap-1.5 text-violet-400 font-medium">
							<span class="h-2 w-2 rounded-full bg-violet-500"></span>
							Views in Period: {formatNumber(period.viewInteractions)}
						</span>
						<span class="flex items-center gap-1.5 text-rose-400 font-medium">
							<span class="h-2 w-2 rounded-full bg-rose-500"></span>
							Likes in Period: {formatNumber(period.likeInteractions)}
						</span>
					</div>
				</div>
			</div>

			<!-- 3. Bottom Two Columns: Top Videos Leaderboard + Category Breakdown -->
			<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
				<!-- Column 1 & 2: Top Performing Videos Table -->
				<div
					class="rounded-3xl border border-border/80 bg-card p-6 shadow-sm lg:col-span-2"
				>
					<div class="mb-5 flex items-center justify-between">
						<div>
							<h2 class="text-lg font-bold text-foreground">Top Performing Content</h2>
							<p class="text-muted-foreground text-xs">
								Ranked by all-time views and viewer interaction.
							</p>
						</div>
						<Button
							href="/dashboard"
							variant="ghost"
							size="sm"
							class="gap-1 rounded-xl text-xs font-semibold text-primary hover:text-primary"
						>
							View All in Studio
							<ChevronRight class="h-3.5 w-3.5" />
						</Button>
					</div>

					{#if (data.topVideos || []).length === 0}
						<div class="py-12 text-center text-sm text-muted-foreground">
							No video metrics recorded yet.
						</div>
					{:else}
						<div class="overflow-x-auto">
							<table class="w-full text-left text-sm">
								<thead>
									<tr class="border-b border-border/60 text-xs font-semibold text-muted-foreground">
										<th class="pb-3 pr-2 w-10">#</th>
										<th class="pb-3 pr-4">Video Details</th>
										<th class="pb-3 pr-4 text-right">Views</th>
										<th class="pb-3 pr-4 text-right">Likes</th>
										<th class="pb-3 text-right">Engagement</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-border/40">
									{#each data.topVideos as video, index}
										{@const viewsPct =
											totalLifetimeViews > 0
												? Math.round((video.views / totalLifetimeViews) * 100)
												: 0}
										{@const engRatio =
											video.views > 0
												? (((video.likesCount || 0) + (video.commentCount || 0)) / video.views) *
													100
												: 0}
										<tr class="group transition-colors hover:bg-muted/30">
											<!-- Rank Medal -->
											<td class="py-3.5 pr-2 font-mono text-xs font-bold text-muted-foreground">
												{#if index === 0}
													<span class="text-amber-400 font-black">🥇 1</span>
												{:else if index === 1}
													<span class="text-slate-300 font-black">🥈 2</span>
												{:else if index === 2}
													<span class="text-amber-600 font-black">🥉 3</span>
												{:else}
													<span>#{index + 1}</span>
												{/if}
											</td>

											<!-- Thumbnail & Title -->
											<td class="py-3.5 pr-4">
												<div class="flex items-center gap-3">
													<a
														href={`/watch/${video.fileId || video.id}`}
														class="relative block h-12 w-20 shrink-0 overflow-hidden rounded-xl bg-muted transition-transform group-hover:scale-105"
													>
														<VideoThumbnail
															{video}
															class="h-full w-full rounded-xl object-cover"
														/>
														{#if video.duration}
															<span
																class="absolute right-1 bottom-1 rounded bg-black/80 px-1 py-0.5 text-[9px] font-bold text-white"
															>
																{formatDuration(video.duration)}
															</span>
														{/if}
													</a>
													<div class="min-w-0 max-w-[280px] sm:max-w-[340px]">
														<a
															href={`/watch/${video.fileId || video.id}`}
															class="line-clamp-1 font-semibold text-foreground hover:text-primary transition-colors text-xs sm:text-sm"
															title={video.title}
														>
															{video.title}
														</a>
														<div class="mt-1 flex items-center gap-2 text-[11px] text-muted-foreground">
															<span class="rounded bg-muted px-1.5 py-0.5 font-medium">
																{video.category || 'General'}
															</span>
															{#if video.createdAt}
																<span>•</span>
																<span>{formatDate(video.createdAt)}</span>
															{/if}
														</div>
													</div>
												</div>
											</td>

											<!-- Views Column -->
											<td class="py-3.5 pr-4 text-right">
												<div class="font-bold text-foreground">{formatNumber(video.views)}</div>
												<div class="text-[11px] text-muted-foreground">{viewsPct}% of total</div>
											</td>

											<!-- Likes Column -->
											<td class="py-3.5 pr-4 text-right font-medium text-foreground">
												<div class="flex items-center justify-end gap-1 text-xs">
													<ThumbsUp class="h-3 w-3 text-rose-500" />
													<span>{formatNumber(video.likesCount)}</span>
												</div>
												<div class="mt-0.5 flex items-center justify-end gap-1 text-[11px] text-muted-foreground">
													<MessageSquare class="h-2.5 w-2.5" />
													<span>{formatNumber(video.commentCount)}</span>
												</div>
											</td>

											<!-- Engagement Rate -->
											<td class="py-3.5 text-right">
												<span
													class="inline-block rounded-full px-2 py-0.5 text-[11px] font-bold {engRatio >=
													5
														? 'bg-emerald-500/15 text-emerald-500'
														: engRatio >= 2
															? 'bg-amber-500/15 text-amber-500'
															: 'bg-muted text-muted-foreground'}"
												>
													{engRatio.toFixed(1)}%
												</span>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}
				</div>

				<!-- Column 3: Category Distribution & Audience Insights -->
				<div class="space-y-6">
					<!-- Category Distribution Card -->
					<div class="rounded-3xl border border-border/80 bg-card p-6 shadow-sm">
						<div class="mb-4 flex items-center justify-between">
							<div>
								<h2 class="text-base font-bold text-foreground">Content Breakdown</h2>
								<p class="text-muted-foreground text-xs">Views and uploads by category</p>
							</div>
							<div class="rounded-xl bg-primary/10 p-2 text-primary">
								<PieChart class="h-4 w-4" />
							</div>
						</div>

						{#if (data.categories || []).length === 0}
							<div class="py-8 text-center text-xs text-muted-foreground">
								No category distribution available yet.
							</div>
						{:else}
							<div class="space-y-4">
								{#each data.categories as cat, idx}
									{@const color = categoryColors[idx % categoryColors.length]}
									{@const catPct =
										totalLifetimeViews > 0
											? Math.round((cat.views / totalLifetimeViews) * 100)
											: 0}
									<div class="space-y-1.5">
										<div class="flex items-center justify-between text-xs font-semibold">
											<span class="flex items-center gap-1.5">
												<span class="h-2 w-2 rounded-full {color.bg}"></span>
												<span class="text-foreground">{cat.category}</span>
												<span class="text-muted-foreground font-normal">({cat.videos} vids)</span>
											</span>
											<span class="text-foreground">{formatNumber(cat.views)} ({catPct}%)</span>
										</div>
										<div class="h-2 w-full overflow-hidden rounded-full bg-muted">
											<div
												class="h-full rounded-full {color.bg} transition-all duration-500"
												style="width: {catPct}%"
											></div>
										</div>
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<!-- Channel Milestone / Growth Tips Card -->
					<div
						class="rounded-3xl border border-border/80 bg-gradient-to-br from-card via-card to-primary/5 p-6 shadow-sm"
					>
						<div class="flex items-center gap-3">
							<div class="rounded-2xl bg-amber-500/15 p-2.5 text-amber-500">
								<Award class="h-5 w-5" />
							</div>
							<div>
								<h3 class="text-sm font-bold text-foreground">Creator Insights</h3>
								<p class="text-muted-foreground text-xs">Audience engagement analysis</p>
							</div>
						</div>

						<div class="mt-4 space-y-3 text-xs">
							<div
								class="flex items-center justify-between rounded-2xl border border-border/60 bg-muted/30 p-3"
							>
								<span class="text-muted-foreground">Upload Frequency</span>
								<span class="font-bold text-foreground">{stats.videos} Total Uploads</span>
							</div>
							<div
								class="flex items-center justify-between rounded-2xl border border-border/60 bg-muted/30 p-3"
							>
								<span class="text-muted-foreground">Overall Engagement</span>
								<span class="font-bold text-emerald-500"
									>{(stats.overallEngagementRate || 0).toFixed(1)}% Active</span
								>
							</div>
							<div
								class="flex items-center justify-between rounded-2xl border border-border/60 bg-muted/30 p-3"
							>
								<span class="text-muted-foreground">Avg Views Per Video</span>
								<span class="font-bold text-foreground"
									>{(stats.avgViewsPerVideo || 0).toFixed(0)} views</span
								>
							</div>
						</div>

						<div class="mt-4 pt-3 border-t border-border/60">
							<Button
								href="/upload"
								variant="outline"
								class="w-full gap-2 rounded-2xl border-primary/30 text-xs font-bold text-primary hover:bg-primary/10"
							>
								<PlusCircle class="h-3.5 w-3.5" />
								Upload Another Video
							</Button>
						</div>
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>
