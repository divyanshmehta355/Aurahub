<script lang="ts">
	import { page } from '$app/state';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { Trash2, Edit2, Check, X, Play, Share2, Globe, Lock } from 'lucide-svelte';
	import { userState } from '#lib/user.svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';

	const queryClient = useQueryClient();

	let id = $derived(page.params.id);
	let isEditing = $state(false);
	let editName = $state('');

	const playlistQuery = createQuery(
		() => ({
			queryKey: ['playlist', id],
			queryFn: async () => {
				const res = await fetchApi(`/playlists/${id}`);
				const playlist = res.playlist || res;
				if (res.videos) playlist.videos = res.videos;
				editName = playlist.title;
				return playlist;
			}
		}),
		() => queryClient
	);

	const updateTitleMutation = createMutation(
		() => ({
			mutationFn: async (title: string) => {
				return await fetchApi(`/playlists/${id}`, {
					method: 'PUT',
					body: JSON.stringify({ title })
				});
			},
			onSuccess: (_, newTitle) => {
				isEditing = false;
				queryClient.setQueryData(['playlist', id], (old: any) => ({ ...old, title: newTitle }));
			}
		}),
		() => queryClient
	);

	const togglePrivacyMutation = createMutation(
		() => ({
			mutationFn: async () => {
				const current = queryClient.getQueryData(['playlist', id]) as any;
				return await fetchApi(`/playlists/${id}`, {
					method: 'PUT',
					body: JSON.stringify({ isPublic: !current.isPublic, title: current.title })
				});
			},
			onSuccess: () => {
				queryClient.setQueryData(['playlist', id], (old: any) => ({
					...old,
					isPublic: !old.isPublic
				}));
			}
		}),
		() => queryClient
	);

	const removeVideoMutation = createMutation(
		() => ({
			mutationFn: async (videoId: string | number) => {
				return await fetchApi(`/playlists/${id}/videos`, {
					method: 'POST',
					body: JSON.stringify({ videoId, action: 'remove' })
				});
			},
			onSuccess: (_, videoId) => {
				queryClient.setQueryData(['playlist', id], (old: any) => ({
					...old,
					videos: old.videos.filter((v: any) => v.id !== videoId && v.fileId !== videoId)
				}));
			}
		}),
		() => queryClient
	);

	async function saveEdit() {
		if (!editName.trim()) return;
		updateTitleMutation.mutate(editName.trim());
	}

	function handleShare() {
		navigator.clipboard.writeText(window.location.href);
		alert('Link copied to clipboard!');
	}

	function formatDuration(seconds: number) {
		if (!seconds) return '0:00';
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
</script>

<svelte:head>
	<title>{playlistQuery.data ? playlistQuery.data.title : 'Playlist'} - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-[1200px] px-4 py-4 md:px-6 md:py-6">
	{#if playlistQuery.isPending}
		<div class="flex flex-col gap-6 md:flex-row md:gap-8">
			<Skeleton class="h-[250px] w-full rounded-xl md:h-[400px] md:w-1/3" />
			<div class="w-full space-y-4 md:w-2/3">
				<Skeleton class="h-20 rounded-xl md:h-24" />
				<Skeleton class="h-20 rounded-xl md:h-24" />
				<Skeleton class="h-20 rounded-xl md:h-24" />
			</div>
		</div>
	{:else if playlistQuery.isError || !playlistQuery.data}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="bg-destructive/10 text-destructive rounded-xl p-4">
				<p>{playlistQuery.error?.message || 'Playlist not found.'}</p>
				<Button variant="outline" class="mt-4" onclick={() => playlistQuery.refetch()}
					>Try again</Button
				>
			</div>
		</div>
	{:else}
		<div class="flex flex-col gap-6 md:flex-row md:gap-8">
			<!-- Playlist Info Sidebar -->
			<div class="w-full shrink-0 md:w-[350px]">
				<div class="bg-card sticky top-20 rounded-2xl border p-4 shadow-sm md:top-24 md:p-6">
					<!-- Thumbnail Stack effect -->
					<div
						class="bg-muted group relative mb-4 flex aspect-video items-center justify-center overflow-hidden rounded-xl shadow-md md:mb-6"
					>
						{#if playlistQuery.data.videos && playlistQuery.data.videos.length > 0}
							<img
								src={playlistQuery.data.videos[0].thumbnailUrl ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${playlistQuery.data.videos[0].title}`}
								alt="Cover"
								class="h-full w-full scale-110 object-cover opacity-50 blur-sm"
							/>
							<img
								src={playlistQuery.data.videos[0].thumbnailUrl ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${playlistQuery.data.videos[0].title}`}
								alt="Cover"
								class="absolute h-full w-4/5 object-cover shadow-lg transition-transform duration-300 group-hover:scale-105"
							/>
						{:else}
							<div class="text-muted-foreground flex flex-col items-center opacity-50">
								<Play class="mb-2 h-10 w-10 md:h-12 md:w-12" />
								<span class="text-sm">Empty</span>
							</div>
						{/if}
						<div
							class="pointer-events-none absolute inset-0 bg-black/20 transition-colors group-hover:bg-black/10"
						></div>
					</div>

					{#if isEditing}
						<div class="mb-4 space-y-3">
							<Input
								type="text"
								bind:value={editName}
								class="h-12 text-lg font-bold md:text-xl"
								autofocus
								disabled={updateTitleMutation.isPending}
							/>
							<div class="flex justify-end gap-2">
								<Button
									variant="ghost"
									size="icon"
									onclick={() => {
										isEditing = false;
										editName = playlistQuery.data.title;
									}}
									disabled={updateTitleMutation.isPending}
								>
									<X class="h-4 w-4" />
								</Button>
								<Button
									size="icon"
									variant="secondary"
									onclick={saveEdit}
									disabled={updateTitleMutation.isPending}
								>
									<Check class="h-4 w-4" />
								</Button>
							</div>
						</div>
					{:else}
						<div class="mb-2 flex items-start justify-between gap-4">
							<h1 class="text-xl font-bold tracking-tight break-words md:text-2xl">
								{playlistQuery.data.title}
							</h1>
							{#if userState.user?.id === playlistQuery.data.ownerId || userState.user?.id === playlistQuery.data.userId}
								<Button
									variant="ghost"
									size="icon"
									onclick={() => (isEditing = true)}
									class="text-muted-foreground h-8 w-8"
								>
									<Edit2 class="h-4 w-4" />
								</Button>
							{/if}
						</div>
					{/if}

					<div class="text-muted-foreground mb-4 space-y-1 text-xs font-medium md:mb-6 md:text-sm">
						<p>{playlistQuery.data.user?.username || 'Unknown User'}</p>
						<p>{playlistQuery.data.videos?.length || 0} videos</p>
					</div>

					{#if playlistQuery.data.videos && playlistQuery.data.videos.length > 0}
						<Button
							href={`/watch/${playlistQuery.data.videos[0].fileId || playlistQuery.data.videos[0].id}?list=${playlistQuery.data.id}`}
							class="h-10 w-full rounded-full text-sm font-semibold shadow-sm md:h-12 md:text-base"
						>
							<Play class="mr-2 h-4 w-4 fill-current md:h-5 md:w-5" />
							Play all
						</Button>
					{/if}

					<div class="mt-4 flex w-full gap-2">
						<Button
							variant="secondary"
							class="h-9 flex-1 rounded-full text-xs font-semibold shadow-sm md:h-10 md:text-sm"
							onclick={handleShare}
						>
							<Share2 class="mr-1.5 h-3.5 w-3.5 md:h-4 md:w-4" /> Share
						</Button>

						{#if userState.user?.id === playlistQuery.data.ownerId || userState.user?.id === playlistQuery.data.userId}
							<Button
								variant="secondary"
								class="h-9 flex-1 rounded-full text-xs font-semibold shadow-sm md:h-10 md:text-sm"
								onclick={() => togglePrivacyMutation.mutate()}
								disabled={togglePrivacyMutation.isPending}
							>
								{#if playlistQuery.data.isPublic}
									<Globe class="mr-1.5 h-3.5 w-3.5 md:h-4 md:w-4" /> Public
								{:else}
									<Lock class="mr-1.5 h-3.5 w-3.5 md:h-4 md:w-4" /> Private
								{/if}
							</Button>
						{/if}
					</div>
				</div>
			</div>

			<!-- Videos List -->
			<div class="flex-1">
				{#if !playlistQuery.data.videos || playlistQuery.data.videos.length === 0}
					<div class="bg-card rounded-2xl border p-8 text-center shadow-sm md:p-12">
						<p class="text-muted-foreground text-base font-medium md:text-lg">
							No videos in this playlist yet.
						</p>
					</div>
				{:else}
					<div class="flex flex-col gap-2 md:gap-3">
						{#each playlistQuery.data.videos as video, i}
							<div
								class="group hover:bg-muted/50 relative flex items-center gap-2 rounded-xl border border-transparent p-2 transition-colors hover:border-border md:gap-4 md:p-3"
							>
								<span
									class="text-muted-foreground w-4 text-center text-xs font-medium md:w-6 md:text-sm"
									>{i + 1}</span
								>

								<a
									href={`/watch/${video.fileId || video.id}?list=${playlistQuery.data.id}`}
									class="flex flex-1 flex-row gap-3 overflow-hidden md:gap-4"
								>
									<div class="w-32 shrink-0 md:w-40">
										<VideoThumbnail {video} class="aspect-video rounded-md md:rounded-lg">
											<div
												class="absolute right-1 bottom-1 rounded bg-black/80 px-1 py-0.5 text-[9px] font-medium text-white backdrop-blur-sm md:right-1.5 md:bottom-1.5 md:text-[10px]"
											>
												{formatDuration(video.duration)}
											</div>
										</VideoThumbnail>
									</div>
									<div class="flex flex-col justify-center overflow-hidden py-1 pr-6 md:pr-0">
										<h3
											class="line-clamp-2 text-sm leading-tight font-semibold transition-colors group-hover:text-primary md:text-base"
										>
											{video.title}
										</h3>
										<p class="text-muted-foreground mt-1 text-[11px] md:text-sm">
											{video.uploader_username || 'Unknown'} • {video.views || 0} views
										</p>
									</div>
								</a>

								{#if userState.user?.id === playlistQuery.data.userId || userState.user?.id === playlistQuery.data.ownerId}
									<Button
										variant="ghost"
										size="icon"
										onclick={() => removeVideoMutation.mutate(video.id || video.fileId)}
										class="text-muted-foreground hover:bg-destructive/10 hover:text-destructive absolute right-2 h-8 w-8 shrink-0 rounded-full opacity-100 transition-all group-hover:opacity-100 md:relative md:right-0 md:opacity-0"
										title="Remove from playlist"
									>
										<Trash2 class="h-4 w-4 md:h-5 md:w-5" />
									</Button>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
