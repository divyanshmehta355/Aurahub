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
		class="flex flex-col justify-between gap-4 border-b border-slate-border/60 pb-4 md:flex-row md:items-end md:border-0 md:pb-0"
	>
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-soft-ivory md:text-3xl font-serif">Creator Studio</h1>
			<p class="mt-1 text-xs text-muted-silver md:text-sm">
				Manage your video catalog, metadata, and visibility settings.
			</p>
		</div>
		<div class="flex items-center gap-2 md:gap-3">
			<Button variant="outline" href="/analytics" class="flex-1 md:flex-none rounded-full border-slate-border text-soft-ivory hover:bg-slate-elevated">
				<BarChart class="mr-2 h-4 w-4 text-aura-lavender" />
				Analytics
			</Button>
			<Button href="/upload" class="flex-1 md:flex-none rounded-full bg-aura-lavender text-midnight-ink font-bold hover:bg-aura-lavender/90 shadow-md shadow-aura-lavender/20">
				<Upload class="mr-2 h-4 w-4" />
				Upload Video
			</Button>
		</div>
	</div>

	<div class="min-h-[400px]">
		{#if dashboardQuery.isPending}
			<div class="space-y-4">
				{#each Array(5) as _}
					<Skeleton class="h-16 w-full rounded-2xl bg-deep-slate/40" />
				{/each}
			</div>
		{:else if dashboardQuery.isError}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<div class="rounded-2xl border border-destructive/30 bg-destructive/10 p-6 text-destructive">
					<p>{dashboardQuery.error.message || 'Failed to load dashboard data.'}</p>
					<Button variant="outline" class="mt-4 rounded-full border-destructive/40" onclick={() => dashboardQuery.refetch()}
						>Try again</Button
					>
				</div>
			</div>
		{:else}
			<div class="space-y-4">
				{#if selectedVideos.size > 0}
					<div class="flex flex-wrap items-center gap-3 rounded-2xl border border-slate-border/80 bg-deep-slate/90 p-3.5 backdrop-blur-xl shadow-lg md:gap-4">
						<span class="text-xs font-bold text-soft-ivory">{selectedVideos.size} selected</span>
						<div class="hidden h-4 w-[1px] bg-slate-border md:block"></div>
						<div class="flex flex-wrap items-center w-full gap-2 md:w-auto">
							<Button
								variant="outline"
								size="sm"
								class="rounded-full border-slate-border text-xs text-soft-ivory hover:bg-slate-elevated"
								onclick={() => handleBulkVisibility('public')}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Make Public
							</Button>
							<Button
								variant="outline"
								size="sm"
								class="rounded-full border-slate-border text-xs text-soft-ivory hover:bg-slate-elevated"
								onclick={() => handleBulkVisibility('private')}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Make Private
							</Button>

							<div class="hidden h-4 w-[1px] bg-slate-border md:block"></div>

							<Button
								variant="outline"
								size="sm"
								class="rounded-full border-destructive/40 text-xs text-destructive hover:bg-destructive/10"
								onclick={() => handleBulkAdult(true)}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Mark as 18+
							</Button>
							<Button
								variant="outline"
								size="sm"
								class="rounded-full border-slate-border text-xs text-soft-ivory hover:bg-slate-elevated"
								onclick={() => handleBulkAdult(false)}
								disabled={bulkVisibilityMutation.isPending || bulkAdultMutation.isPending}
							>
								Mark General
							</Button>

							<Button
								variant="destructive"
								size="sm"
								class="rounded-full text-xs md:ml-auto"
								onclick={handleBulkDelete}
								disabled={bulkDeleteMutation.isPending}
							>
								<Trash2 class="h-3.5 w-3.5 md:mr-1.5" />
								<span class="hidden md:inline">Delete</span>
							</Button>
						</div>
					</div>
				{/if}

				<div
					class="w-[calc(100vw-2rem)] overflow-x-auto rounded-3xl border border-slate-border/70 bg-deep-slate/40 shadow-xl backdrop-blur-xl md:w-full"
				>
					<table class="w-full text-left text-sm whitespace-nowrap md:whitespace-normal">
						<thead class="bg-midnight-ink/60 text-muted-silver border-b border-slate-border/60 text-xs font-semibold">
							<tr>
								<th class="w-12 p-3.5 text-center">
									<input
										type="checkbox"
										class="h-4 w-4 rounded border-slate-border/80 text-aura-lavender focus:ring-aura-lavender bg-midnight-ink"
										checked={dashboardQuery.data.videos.length > 0 &&
											selectedVideos.size === dashboardQuery.data.videos.length}
										onchange={toggleAll}
									/>
								</th>
								<th class="p-3.5 font-medium">Video</th>
								<th class="p-3.5 font-medium">Category & Tags</th>
								<th class="p-3.5 font-medium">Visibility</th>
								<th class="p-3.5 font-medium">Views</th>
								<th class="p-3.5 font-medium">Date</th>
								<th class="p-3.5 text-right font-medium">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-slate-border/30">
							{#if dashboardQuery.data.videos.length === 0}
								<tr>
									<td colspan="7" class="p-8 text-center text-base text-muted-silver">
										No videos uploaded yet.
									</td>
								</tr>
							{/if}
							{#each dashboardQuery.data.videos as video}
								<tr class="hover:bg-slate-elevated/40 transition-colors">
									<td class="p-3 text-center">
										<input
											type="checkbox"
											class="h-4 w-4 rounded border-slate-border/80 text-aura-lavender focus:ring-aura-lavender bg-midnight-ink"
											checked={selectedVideos.has(video.id)}
											onchange={() => toggleSelection(video.id)}
										/>
									</td>
									<td class="p-3">
										<div class="flex min-w-[200px] max-w-xs md:max-w-md items-center gap-3">
											<VideoThumbnail
												{video}
												class="h-10 w-16 shrink-0 rounded-xl overflow-hidden border border-slate-border/40 md:h-12 md:w-20"
											/>
											<div class="flex flex-col overflow-hidden min-w-0 flex-1">
												<span class="line-clamp-1 text-sm font-semibold md:text-base text-soft-ivory break-words [overflow-wrap:anywhere] truncate" title={video.title}>
													{video.title}
												</span>
												<span class="text-muted-silver/70 text-[11px] font-mono md:text-xs">
													{formatDuration(video.duration)}
												</span>
											</div>
										</div>
									</td>
									<td class="p-3">
										<div class="flex flex-col gap-1 max-w-[180px]">
											<span class="text-xs font-semibold text-soft-ivory">
												{video.category || 'Other'}
											</span>
											{#if video.tags && video.tags.length > 0}
												<div class="flex flex-wrap gap-1">
													{#each video.tags.slice(0, 2) as tag}
														<span class="rounded-md border border-aura-lavender/30 bg-aura-lavender/10 px-1.5 py-0.5 text-[10px] font-semibold text-aura-lavender">
															#{tag.replace(/^#+/, '')}
														</span>
													{/each}
													{#if video.tags.length > 2}
														<span class="text-[10px] text-muted-silver">
															+{video.tags.length - 2}
														</span>
													{/if}
												</div>
											{:else}
												<span class="text-[11px] text-muted-silver/60">No tags</span>
											{/if}
										</div>
									</td>
									<td class="p-3">
										<div class="flex flex-wrap items-center gap-1.5">
											<span
												class="inline-flex items-center rounded-full border border-slate-border/80 bg-slate-elevated px-2 py-0.5 text-xs font-medium text-soft-ivory"
											>
												{video.visibility || 'public'}
											</span>
											{#if video.isAdult}
												<span
													class="inline-flex items-center rounded-full border border-destructive/30 bg-destructive/15 px-2 py-0.5 text-[10px] font-semibold text-destructive"
												>
													18+ Adult
												</span>
											{/if}
										</div>
									</td>
									<td class="p-3 font-mono text-muted-silver text-xs">{video.views || 0}</td>
									<td class="p-3 text-xs text-muted-silver md:text-sm">
										{new Date(video.createdAt || video.created_at || Date.now()).toLocaleDateString()}
									</td>
									<td class="p-3 text-right">
										<Button
											variant="outline"
											size="sm"
											class="h-8 gap-1.5 rounded-full border-slate-border/80 px-3 text-xs font-semibold text-soft-ivory hover:border-aura-lavender/50 hover:bg-aura-lavender/10 hover:text-aura-lavender transition-all shadow-sm"
											onclick={() => openEditModal(video)}
											title="Edit video metadata"
										>
											<Edit class="h-3.5 w-3.5" />
											<span>Edit</span>
										</Button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Pagination Controls -->
				<div class="mt-8 flex items-center justify-between border-t border-slate-border/60 pt-6 pb-4">
					<Button
						variant="outline"
						onclick={() => changePage(currentPage - 1)}
						disabled={currentPage === 1 || dashboardQuery.isFetching}
						class="rounded-full border-slate-border text-soft-ivory px-6 hover:bg-slate-elevated disabled:opacity-40"
					>
						Previous
					</Button>
					<span class="text-xs font-semibold text-muted-silver">
						Page {currentPage}
					</span>
					<Button
						variant="default"
						onclick={() => changePage(currentPage + 1)}
						disabled={!dashboardQuery.data.hasMore || dashboardQuery.isFetching}
						class="rounded-full bg-aura-lavender text-midnight-ink font-semibold px-6 hover:bg-aura-lavender/90 disabled:opacity-40"
					>
						Next
					</Button>
				</div>
			</div>
		{/if}
	</div>
</div>

<EditVideoModal bind:open={editModalOpen} video={editingVideo} />
