<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { BarChart, Upload, Trash2, Edit } from 'lucide-svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import EditVideoModal from '#lib/components/EditVideoModal.svelte';
	import { Button } from '#lib/components/ui/button';
	import { Skeleton } from '#lib/components/ui/skeleton';

	const queryClient = useQueryClient();

	let selectedVideos: Set<string> = $state(new Set());
	let currentPage = $state(1);
	let editingVideo: any = $state(null);
	let editModalOpen = $state(false);

	const dashboardQuery = createQuery(
		() => ({
			queryKey: ['dashboard', currentPage],
			queryFn: async () => {
				if (!userState.user && userState.isLoaded) {
					window.location.href = '/login';
					return { videos: [], hasMore: false };
				}
				const dashRes = await fetchApi(`/creator/dashboard?limit=10&page=${currentPage}`).catch(
					() => ({ videos: [], hasMore: false })
				);
				return { videos: dashRes?.videos || [], hasMore: dashRes?.hasMore || false };
			}
		}),
		() => queryClient
	);

	const bulkDeleteMutation = createMutation(
		() => ({
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
		}),
		() => queryClient
	);

	const bulkVisibilityMutation = createMutation(
		() => ({
			mutationFn: async ({ videoIds, visibility }: { videoIds: string[]; visibility: string }) => {
				return await fetchApi('/creator/videos/bulk', {
					method: 'PUT',
					body: JSON.stringify({ videoIds, visibility })
				});
			},
			onSuccess: () => {
				selectedVideos = new Set();
				queryClient.invalidateQueries({ queryKey: ['dashboard'] });
			}
		}),
		() => queryClient
	);

	const bulkAdultMutation = createMutation(
		() => ({
			mutationFn: async ({ videoIds, isAdult }: { videoIds: string[]; isAdult: boolean }) => {
				return await fetchApi('/creator/videos/bulk-adult', {
					method: 'PUT',
					body: JSON.stringify({ videoIds, isAdult })
				});
			},
			onSuccess: () => {
				selectedVideos = new Set();
				queryClient.invalidateQueries({ queryKey: ['dashboard'] });
			}
		}),
		() => queryClient
	);

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
		const videos = dashboardQuery.data?.videos || [];
		if (selectedVideos.size === videos.length) {
			selectedVideos = new Set();
		} else {
			selectedVideos = new Set(videos.map((v: any) => v.id));
		}
	}

	function changePage(newPage: number) {
		if (newPage < 1 || (!dashboardQuery.data?.hasMore && newPage > currentPage)) return;
		currentPage = newPage;
		selectedVideos = new Set(); // Clear selection on page change
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	function openEditModal(video: any) {
		editingVideo = video;
		editModalOpen = true;
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

	function handleBulkAdult(isAdult: boolean) {
		if (selectedVideos.size === 0) return;
		bulkAdultMutation.mutate({ videoIds: Array.from(selectedVideos), isAdult });
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

<div class="mx-auto max-w-6xl space-y-6 px-4 py-4 md:px-6 md:py-6">
	<div
		class="flex flex-col justify-between gap-4 border-b pb-4 md:flex-row md:items-end md:border-0 md:pb-0"
	>
		<div>
			<h1 class="text-2xl font-bold tracking-tight md:text-3xl">Creator Studio</h1>
			<p class="text-muted-foreground mt-1 text-sm md:text-base">
				Manage your videos and upload content.
			</p>
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
				<div class="bg-destructive/10 text-destructive rounded-xl p-4">
					<p>{dashboardQuery.error.message || 'Failed to load dashboard data.'}</p>
					<Button variant="outline" class="mt-4" onclick={() => dashboardQuery.refetch()}
						>Try again</Button
					>
				</div>
			</div>
		{:else}
			<div class="space-y-4">
				{#if selectedVideos.size > 0}
					<div class="bg-muted/50 flex flex-wrap items-center gap-3 rounded-lg border p-3 md:gap-4">
						<span class="text-sm font-medium">{selectedVideos.size} selected</span>
						<div class="hidden h-4 w-[1px] bg-border md:block"></div>
						<div class="flex flex-wrap items-center w-full gap-2 md:w-auto">
							<Button
								variant="secondary"
								size="sm"
								onclick={() => handleBulkVisibility('public')}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Make Public
							</Button>
							<Button
								variant="secondary"
								size="sm"
								onclick={() => handleBulkVisibility('private')}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Make Private
							</Button>

							<div class="hidden h-4 w-[1px] bg-border md:block"></div>

							<Button
								variant="outline"
								size="sm"
								class="border-destructive/40 text-destructive hover:bg-destructive/10"
								onclick={() => handleBulkAdult(true)}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Mark as 18+
							</Button>
							<Button
								variant="secondary"
								size="sm"
								onclick={() => handleBulkAdult(false)}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Mark as General
							</Button>

							<Button
								variant="destructive"
								size="sm"
								class="md:ml-auto"
								onclick={handleBulkDelete}
								disabled={bulkDeleteMutation.isPending}
							>
								<Trash2 class="h-4 w-4 md:mr-2" />
								<span class="hidden md:inline">Delete</span>
							</Button>
						</div>
					</div>
				{/if}

				<div
					class="bg-card w-[calc(100vw-2rem)] overflow-x-auto rounded-lg border shadow-sm md:w-full"
				>
					<table class="w-full text-left text-sm whitespace-nowrap md:whitespace-normal">
						<thead class="bg-muted/50 text-muted-foreground border-b">
							<tr>
								<th class="w-12 p-3 text-center">
									<input
										type="checkbox"
										class="h-4 w-4 rounded border-border text-primary focus:ring-primary"
										checked={dashboardQuery.data.videos.length > 0 &&
											selectedVideos.size === dashboardQuery.data.videos.length}
										onchange={toggleAll}
									/>
								</th>
								<th class="p-3 font-medium">Video</th>
								<th class="p-3 font-medium">Visibility</th>
								<th class="p-3 font-medium">Views</th>
								<th class="p-3 font-medium">Date</th>
								<th class="p-3"></th>
							</tr>
						</thead>
						<tbody>
							{#if dashboardQuery.data.videos.length === 0}
								<tr>
									<td colspan="5" class="text-muted-foreground p-8 text-center text-base">
										No videos uploaded yet.
									</td>
								</tr>
							{/if}
							{#each dashboardQuery.data.videos as video}
								<tr class="hover:bg-muted/30 border-b transition-colors last:border-0">
									<td class="p-3 text-center">
										<input
											type="checkbox"
											class="h-4 w-4 rounded border-border text-primary focus:ring-primary"
											checked={selectedVideos.has(video.id)}
											onchange={() => toggleSelection(video.id)}
										/>
									</td>
									<td class="p-3">
										<div class="flex min-w-[200px] items-center gap-3">
											<VideoThumbnail
												{video}
												class="bg-muted h-10 w-16 shrink-0 rounded md:h-12 md:w-20"
											/>
											<div class="flex flex-col overflow-hidden">
												<span class="line-clamp-1 text-sm font-medium md:text-base"
													>{video.title}</span
												>
												<span class="text-muted-foreground text-[11px] md:text-xs">
													{formatDuration(video.duration)}
												</span>
											</div>
										</div>
									</td>
									<td class="p-3">
										<div class="flex flex-wrap items-center gap-1.5">
											<span
												class="bg-secondary text-secondary-foreground inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
											>
												{video.visibility || 'public'}
											</span>
											{#if video.isAdult}
												<span
													class="inline-flex items-center rounded-md border border-destructive/20 bg-destructive/15 px-1.5 py-0.5 text-[10px] font-semibold text-destructive"
												>
													18+ Adult
												</span>
											{/if}
										</div>
									</td>
									<td class="text-muted-foreground p-3">{video.views || 0}</td>
									<td class="text-muted-foreground p-3 text-xs md:text-sm">
										{new Date(video.createdAt || video.created_at || Date.now()).toLocaleDateString()}
									</td>
									<td class="p-3 text-right">
										<Button variant="ghost" size="sm" onclick={() => openEditModal(video)}>
											<Edit class="h-4 w-4" />
										</Button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Pagination Controls -->
				<div class="mt-8 flex items-center justify-between border-t border-border pt-6 pb-4">
					<Button
						variant="secondary"
						onclick={() => changePage(currentPage - 1)}
						disabled={currentPage === 1 || dashboardQuery.isFetching}
						class="rounded-full px-6"
					>
						Previous
					</Button>
					<span class="text-muted-foreground text-sm font-medium">
						Page {currentPage}
					</span>
					<Button
						variant="default"
						onclick={() => changePage(currentPage + 1)}
						disabled={!dashboardQuery.data.hasMore || dashboardQuery.isFetching}
						class="rounded-full px-6"
					>
						Next
					</Button>
				</div>
			</div>
		{/if}
	</div>
</div>

<EditVideoModal bind:open={editModalOpen} video={editingVideo} />
