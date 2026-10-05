<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { BarChart, Video } from 'lucide-svelte';

	let analytics: any = $state(null);
	let isLoading = $state(true);

	onMount(async () => {
		if (!userState.user && userState.isLoaded) {
			window.location.href = '/login';
			return;
		}
		await loadAnalyticsData();
	});

	async function loadAnalyticsData() {
		isLoading = true;
		try {
			analytics = await fetchApi('/creator/analytics').catch(() => null);
		} catch (err) {
			console.error('Failed to load analytics data', err);
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Analytics - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-6xl space-y-8">
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Channel Analytics</h1>
			<p class="text-muted-foreground mt-1">
				View your channel performance and growth.
			</p>
		</div>
		<a
			href="/dashboard"
			class="text-primary-foreground inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium transition-colors hover:bg-primary/90"
		>
			<Video class="mr-2 h-4 w-4" />
			Manage Content
		</a>
	</div>

	<div class="min-h-[400px]">
		{#if isLoading}
			<div class="flex h-48 items-center justify-center">
				<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
			</div>
		{:else if analytics}
			<div class="mb-8 grid grid-cols-1 gap-6 md:grid-cols-3">
				<div class="bg-card rounded-xl border p-6">
					<h3 class="text-muted-foreground mb-2 text-sm font-medium">Total Views</h3>
					<p class="text-3xl font-bold">{analytics.total_views || 0}</p>
				</div>
				<div class="bg-card rounded-xl border p-6">
					<h3 class="text-muted-foreground mb-2 text-sm font-medium">Total Likes</h3>
					<p class="text-3xl font-bold">{analytics.total_likes || 0}</p>
				</div>
				<div class="bg-card rounded-xl border p-6">
					<h3 class="text-muted-foreground mb-2 text-sm font-medium">Total Comments</h3>
					<p class="text-3xl font-bold">{analytics.total_comments || 0}</p>
				</div>
			</div>

			<div class="bg-card flex min-h-[200px] items-center justify-center rounded-xl border p-6">
				<p class="text-muted-foreground text-sm">
					More detailed analytics charts will appear here as your channel grows.
				</p>
			</div>
		{:else}
			<div class="text-muted-foreground bg-card rounded-xl border p-8 text-center">
				No analytics data available yet. Upload videos to get started!
			</div>
		{/if}
	</div>
</div>
