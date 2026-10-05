<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { BarChart, Upload, Trash2 } from 'lucide-svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { Button } from '#lib/components/ui/button';
	import { Skeleton } from '#lib/components/ui/skeleton';

	const queryClient = useQueryClient();

	let selectedVideos: Set<string> = $state(new Set());

	const dashboardQuery = createQuery(() => ({
		queryKey: ['dashboard'],
		queryFn: async () => {
			if (!userState.user && userState.isLoaded) {
				window.location.href = '/login';
				return [];
			}
			const dashRes = await fetchApi('/creator/dashboard?limit=50').catch(() => ({ videos: [] }));
			return dashRes?.videos || [];
		}
	}), () => queryClient);

	const bulkDeleteMutation = createMutation(() => ({
		mutationFn: async (videoIds: string[]) => {
			return await fetchApi('/creator/videos/bulk', {
				method: 'DELETE',
				body: JSON.stringify({ videoIds })
			});
		},
		onSuccess: () => {
			selectedVideos = new Set();
			queryClient.invalidateQueries({ queryKey: ['dashboard'] });
		}
	}), () => queryClient);

	const bulkVisibilityMutation = createMutation(() => ({
		mutationFn: async ({ videoIds, visibility }: { videoIds: string[], visibility: string }) => {
			return await fetchApi('/creator/videos/bulk', {
				method: 'PUT',
				body: JSON.stringify({ videoIds, visibility })
			});
		},
		onSuccess: () => {
			selectedVideos = new Set();
			queryClient.invalidateQueries({ queryKey: ['dashboard'] });
		}
	}), () => queryClient);

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
		const videos = dashboardQuery.data || [];
		if (selectedVideos.size === videos.length) {
			selectedVideos = new Set();
		} else {
			selectedVideos = new Set(videos.map((v: any) => v.id));
		}
	}

	function handleBulkDelete() {
		if (selectedVideos.size === 0) return;
		if (!confirm(`Are you sure you want to delete ${selectedVideos.size} videos?`)) return;
		bulkDeleteMutation.mutate(Array.from(selectedVideos));
	}

	function handleBulkVisibility(visibility: string) {
		if (selectedVideos.size === 0) return;
		bulkVisibilityMutation.mutate({ videoIds: Array.from(selectedVideos), visibility });
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

<div class="mx-auto max-w-6xl space-y-6 px-4 md:px-6 py-4 md:py-6">
	<div class="flex flex-col justify-between gap-4 md:flex-row md:items-end border-b pb-4 md:border-0 md:pb-0">
		<div>
			<h1 class="text-2xl md:text-3xl font-bold tracking-tight">Creator Studio</h1>
			<p class="text-muted-foreground mt-1 text-sm md:text-base">Manage your videos and upload content.</p>
		</div>
		<div class="flex items-center gap-2 md:gap-3">
			<Button variant="outline" href="/analytics" class="flex-1 md:flex-none">
				<BarChart class="mr-2 h-4 w-4" />
				Analytics
			</Button>
			<Button href="/upload" class="flex-1 md:flex-none">
				<Upload class="mr-2 h-4 w-4" />
				Upload Video
			</Button>
		</div>
	</div>

	<div class="min-h-[400px]">
		{#if dashboardQuery.isPending}
			<div class="space-y-4">
				{#each Array(5) as _}
					<Skeleton class="h-16 w-full rounded-lg" />
				{/each}
			</div>
		{:else if dashboardQuery.isError}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<div class="rounded-xl bg-destructive/10 p-4 text-destructive">
					<p>{dashboardQuery.error.message || 'Failed to load dashboard data.'}</p>
					<Button variant="outline" class="mt-4" onclick={() => dashboardQuery.refetch()}>Try again</Button>
				</div>
			</div>
		{:else}
			<div class="space-y-4">
				{#if selectedVideos.size > 0}
					<div class="bg-muted/50 flex flex-wrap items-center gap-3 md:gap-4 rounded-lg border p-3">
						<span class="text-sm font-medium">{selectedVideos.size} selected</span>
						<div class="hidden md:block h-4 w-[1px] bg-border"></div>
						<div class="flex gap-2 w-full md:w-auto">
							<Button
								variant="secondary"
								size="sm"
								class="flex-1 md:flex-none"
								onclick={() => handleBulkVisibility('public')}
								disabled={bulkVisibilityMutation.isPending}
							>
								Make Public
							</Button>
							<Button
								variant="secondary"
								size="sm"
								class="flex-1 md:flex-none"
								onclick={() => handleBulkVisibility('private')}
								disabled={bulkVisibilityMutation.isPending}
							>
								Make Private
							</Button>
							<Button
								variant="destructive"
								size="sm"
								class="flex-1 md:flex-none md:ml-auto"
								onclick={handleBulkDelete}
								disabled={bulkDeleteMutation.isPending}
							>
								<Trash2 class="md:mr-2 h-4 w-4" />
								<span class="hidden md:inline">Delete</span>
							</Button>
						</div>
					</div>
				{/if}

				<div class="bg-card overflow-x-auto rounded-lg border shadow-sm w-[calc(100vw-2rem)] md:w-full">
					<table class="w-full text-left text-sm whitespace-nowrap md:whitespace-normal">
						<thead class="bg-muted/50 text-muted-foreground border-b">
							<tr>
								<th class="w-12 p-3 text-center">
									<input
										type="checkbox"
										class="rounded border-border text-primary focus:ring-primary h-4 w-4"
										checked={dashboardQuery.data.length > 0 && selectedVideos.size === dashboardQuery.data.length}
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
							{#if dashboardQuery.data.length === 0}
								<tr>
									<td colspan="5" class="text-muted-foreground p-8 text-center text-base">
										No videos uploaded yet.
									</td>
								</tr>
							{/if}
							{#each dashboardQuery.data as video}
								<tr class="hover:bg-muted/30 border-b transition-colors last:border-0">
									<td class="p-3 text-center">
										<input
											type="checkbox"
											class="rounded border-border text-primary focus:ring-primary h-4 w-4"
											checked={selectedVideos.has(video.id)}
											onchange={() => toggleSelection(video.id)}
										/>
									</td>
									<td class="p-3">
										<div class="flex items-center gap-3 min-w-[200px]">
											<VideoThumbnail {video} class="bg-muted h-10 w-16 md:h-12 md:w-20 shrink-0 rounded" />
											<div class="flex flex-col overflow-hidden">
												<span class="line-clamp-1 font-medium text-sm md:text-base">{video.title}</span>
												<span class="text-muted-foreground text-[11px] md:text-xs">
													{formatDuration(video.duration)}
												</span>
											</div>
										</div>
									</td>
									<td class="p-3">
										<span
											class="bg-secondary text-secondary-foreground inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
										>
											{video.visibility || 'public'}
										</span>
									</td>
									<td class="text-muted-foreground p-3">{video.views || 0}</td>
									<td class="text-muted-foreground p-3 text-xs md:text-sm">
										{new Date(video.created_at).toLocaleDateString()}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	</div>
</div>
