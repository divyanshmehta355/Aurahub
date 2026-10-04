<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { getFallbackThumbnailUrl } from '#lib/thumbnailSvg';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { BarChart, Video, Upload, Edit, Trash2, CheckSquare } from 'lucide-svelte';

	let activeTab = $state('videos');

	let dashboardVideos: any[] = $state([]);
	let analytics: any = $state(null);
	let isLoading = $state(true);

	// Bulk selection state
	let selectedVideos: Set<string> = $state(new Set());

	onMount(async () => {
		// Ensure user is logged in
		if (!userState.user && userState.isLoaded) {
			window.location.href = '/login';
			return;
		}

		await loadDashboardData();
	});

	async function loadDashboardData() {
		isLoading = true;
		try {
			const [dashRes, analyticsRes] = await Promise.all([
				fetchApi('/creator/dashboard?limit=50').catch(() => ({ videos: [] })),
				fetchApi('/creator/analytics').catch(() => null)
			]);

			dashboardVideos = dashRes?.videos || [];
			analytics = analyticsRes;
		} catch (err) {
			console.error('Failed to load dashboard data', err);
		} finally {
			isLoading = false;
		}
	}

	function toggleSelection(id: string) {
		const newSet = new Set(selectedVideos);
		if (newSet.has(id)) {
			newSet.delete(id);
		} else {
			newSet.add(id);
		}
		selectedVideos = newSet;
	}

	function toggleAll() {
		if (selectedVideos.size === dashboardVideos.length) {
			selectedVideos = new Set();
		} else {
			selectedVideos = new Set(dashboardVideos.map((v) => v.id));
		}
	}

	async function handleBulkDelete() {
		if (selectedVideos.size === 0) return;
		if (!confirm(`Are you sure you want to delete ${selectedVideos.size} videos?`)) return;

		try {
			await fetchApi('/creator/videos/bulk', {
				method: 'DELETE',
				body: JSON.stringify({ videoIds: Array.from(selectedVideos) })
			});
			selectedVideos = new Set();
			await loadDashboardData();
		} catch (err) {
			alert('Failed to delete videos');
		}
	}

	async function handleBulkVisibility(visibility: string) {
		if (selectedVideos.size === 0) return;

		try {
			await fetchApi('/creator/videos/bulk', {
				method: 'PUT',
				body: JSON.stringify({
					videoIds: Array.from(selectedVideos),
					visibility: visibility
				})
			});
			selectedVideos = new Set();
			await loadDashboardData();
		} catch (err) {
			alert('Failed to update visibility');
		}
	}

	function formatDuration(seconds: number) {
		if (!seconds) return '0:00';
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
</script>

<svelte:head>
	<title>Creator Dashboard - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-6xl space-y-8">
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Creator Studio</h1>
			<p class="text-muted-foreground mt-1">
				Manage your videos, view analytics, and upload content.
			</p>
		</div>
		<a
			href="/upload"
			class="text-primary-foreground inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium transition-colors hover:bg-primary/90"
		>
			<Upload class="mr-2 h-4 w-4" />
			Upload Video
		</a>
	</div>

	<!-- Custom Tabs (No Sidebar) -->
	<div class="flex border-b">
		<button
			class="border-b-2 px-4 py-3 text-sm font-medium transition-colors {activeTab === 'videos'
				? 'border-primary text-primary'
				: 'text-muted-foreground border-transparent hover:border-border hover:text-primary'}"
			onclick={() => (activeTab = 'videos')}
		>
			<div class="flex items-center gap-2">
				<Video class="h-4 w-4" />
				Content
			</div>
		</button>
		<button
			class="border-b-2 px-4 py-3 text-sm font-medium transition-colors {activeTab === 'analytics'
				? 'border-primary text-primary'
				: 'text-muted-foreground border-transparent hover:border-border hover:text-primary'}"
			onclick={() => (activeTab = 'analytics')}
		>
			<div class="flex items-center gap-2">
				<BarChart class="h-4 w-4" />
				Analytics
			</div>
		</button>
	</div>

	<!-- Tab Content -->
	<div class="min-h-[400px]">
		{#if isLoading}
			<div class="flex h-48 items-center justify-center">
				<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
			</div>
		{:else if activeTab === 'videos'}
			<!-- Videos Tab -->
			<div class="space-y-4">
				{#if selectedVideos.size > 0}
					<div class="bg-muted/50 flex items-center gap-4 rounded-lg border p-3">
						<span class="text-sm font-medium">{selectedVideos.size} selected</span>
						<div class="h-4 w-[1px] bg-border"></div>
						<button
							onclick={() => handleBulkVisibility('public')}
							class="text-sm font-medium hover:text-primary">Make Public</button
						>
						<button
							onclick={() => handleBulkVisibility('private')}
							class="text-sm font-medium hover:text-primary">Make Private</button
						>
						<button
							onclick={handleBulkDelete}
							class="ml-auto flex items-center gap-1 text-sm font-medium text-red-500 hover:text-red-600"
						>
							<Trash2 class="h-4 w-4" />
							Delete
						</button>
					</div>
				{/if}

				<div class="bg-card overflow-hidden rounded-lg border">
					<table class="w-full text-left text-sm">
						<thead class="bg-muted/50 text-muted-foreground border-b">
							<tr>
								<th class="w-12 p-3 text-center">
									<input
										type="checkbox"
										class="rounded border-border text-primary focus:ring-primary"
										checked={dashboardVideos.length > 0 &&
											selectedVideos.size === dashboardVideos.length}
										onchange={toggleAll}
									/>
								</th>
								<th class="p-3 font-medium">Video</th>
								<th class="p-3 font-medium">Visibility</th>
								<th class="p-3 font-medium">Views</th>
								<th class="p-3 font-medium">Date</th>
							</tr>
						</thead>
						<tbody>
							{#if dashboardVideos.length === 0}
								<tr>
									<td colspan="5" class="text-muted-foreground p-8 text-center">
										No videos uploaded yet.
									</td>
								</tr>
							{/if}
							{#each dashboardVideos as video}
								<tr class="hover:bg-muted/30 border-b transition-colors last:border-0">
									<td class="p-3 text-center">
										<input
											type="checkbox"
											class="rounded border-border text-primary focus:ring-primary"
											checked={selectedVideos.has(video.id)}
											onchange={() => toggleSelection(video.id)}
										/>
									</td>
									<td class="p-3">
										<div class="flex items-center gap-3">
											<VideoThumbnail 
												{video}
												class="bg-muted h-12 w-20 shrink-0 rounded"
											/>
											<div class="flex flex-col">
												<span class="line-clamp-1 font-medium">{video.title}</span>
												<span class="text-muted-foreground text-xs"
													>{formatDuration(video.duration)}</span
												>
											</div>
										</div>
									</td>
									<td class="p-3">
										<span
											class="bg-muted text-muted-foreground inline-flex items-center rounded px-2 py-0.5 text-xs font-medium"
										>
											{video.visibility || 'public'}
										</span>
									</td>
									<td class="text-muted-foreground p-3">{video.views || 0}</td>
									<td class="text-muted-foreground p-3 text-xs"
										>{new Date(video.created_at).toLocaleDateString()}</td
									>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{:else if activeTab === 'analytics'}
			<!-- Analytics Tab -->
			{#if analytics}
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
		{/if}
	</div>
</div>
