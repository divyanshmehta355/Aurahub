<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { ThumbsUp, Share2, MessageSquare, Play, ListPlus, Loader2 } from 'lucide-svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import SaveToPlaylistModal from '#lib/components/SaveToPlaylistModal.svelte';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	
	import { createQuery, createMutation, createInfiniteQuery, useQueryClient } from '@tanstack/svelte-query';

	const queryClient = useQueryClient();

	let videoId = $derived(page.params.id);
	let playlistId = $derived(page.url.searchParams.get('list'));
	let isSaveModalOpen = $state(false);
	let newCommentText = $state('');

	// Queries
	const videoQuery = createQuery(() => ({
		queryKey: ['video', videoId],
		queryFn: async () => {
			const data = await fetchApi(`/videos/${videoId}`);
			// Record view async
			fetchApi(`/videos/${data.id || data.fileId || videoId}/view`, { method: 'POST' }).catch(console.error);
			return data;
		}
	}), () => queryClient);

	let uploaderUsername = $derived(videoQuery.data?.uploader?.username);
	const uploaderQuery = createQuery(() => ({
		queryKey: ['profile', uploaderUsername],
		queryFn: async () => await fetchApi(`/users/${uploaderUsername}/profile`),
		enabled: !!uploaderUsername
	}), () => queryClient);

	const playlistQuery = createQuery(() => ({
		queryKey: ['playlist', playlistId],
		queryFn: async () => {
			const res = await fetchApi(`/playlists/${playlistId}`);
			const p = res.playlist || res;
			if (res.videos) p.videos = res.videos;
			return p;
		},
		enabled: !!playlistId
	}), () => queryClient);

	const commentsQuery = createQuery(() => ({
		queryKey: ['comments', videoId],
		queryFn: async () => {
			const res = await fetchApi(`/comments/${videoId}`);
			return res.comments || res.data || [];
		}
	}), () => queryClient);

	const suggestionsQuery = createInfiniteQuery(() => ({
		queryKey: ['suggestions', videoId],
		queryFn: async ({ pageParam = 1 }) => {
			const res = await fetchApi(`/videos/suggestions?exclude=${videoId}&limit=10&page=${pageParam}`);
			return res;
		},
		getNextPageParam: (lastPage: any) => {
			return lastPage.currentPage < lastPage.totalPages ? lastPage.currentPage + 1 : undefined;
		},
		initialPageParam: 1
	}), () => queryClient);

	// Infinite scroll action
	function infiniteScroll(node: HTMLElement) {
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries[0].isIntersecting) {
					if (suggestionsQuery.hasNextPage && !suggestionsQuery.isFetchingNextPage) {
						suggestionsQuery.fetchNextPage();
					}
				}
			},
			{ rootMargin: '200px' }
		);
		observer.observe(node);
		return { destroy() { observer.disconnect(); } };
	}

	// Mutations
	const likeMutation = createMutation(() => ({
		mutationFn: async (action: 'like' | 'unlike') => {
			return await fetchApi(`/videos/${videoId}/like?action=${action}`, { method: 'POST' });
		},
		onMutate: async (action) => {
			await queryClient.cancelQueries({ queryKey: ['video', videoId] });
			const previous = queryClient.getQueryData(['video', videoId]);
			queryClient.setQueryData(['video', videoId], (old: any) => ({
				...old,
				isLiked: action === 'like',
				likesCount: old.likesCount + (action === 'like' ? 1 : -1)
			}));
			return { previous };
		},
		onError: (err, newTodo, context: any) => {
			queryClient.setQueryData(['video', videoId], context?.previous);
		},
		onSettled: () => {
			queryClient.invalidateQueries({ queryKey: ['video', videoId] });
		}
	}), () => queryClient);

	const subscribeMutation = createMutation(() => ({
		mutationFn: async () => {
			return await fetchApi(`/users/${uploaderUsername}/subscribe`, { method: 'POST' });
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['profile', uploaderUsername] });
		}
	}), () => queryClient);

	const commentMutation = createMutation(() => ({
		mutationFn: async (content: string) => {
			return await fetchApi('/comments', {
				method: 'POST',
				body: JSON.stringify({ videoId, content })
			});
		},
		onSuccess: () => {
			newCommentText = '';
			queryClient.invalidateQueries({ queryKey: ['comments', videoId] });
		}
	}), () => queryClient);

	// Handlers
	function handleLike() {
		if (!userState.user) return (window.location.href = '/login');
		likeMutation.mutate(videoQuery.data?.isLiked ? 'unlike' : 'like');
	}

	function handleSubscribe() {
		if (!userState.user) return (window.location.href = '/login');
		subscribeMutation.mutate();
	}

	function handleCommentSubmit() {
		if (!newCommentText.trim() || !userState.user) return;
		commentMutation.mutate(newCommentText.trim());
	}

	function handleShare() {
		navigator.clipboard.writeText(window.location.href);
		alert('Link copied to clipboard!');
	}

	function formatTimeAgo(dateString: string) {
		const date = new Date(dateString);
		const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
		let interval = seconds / 31536000;
		if (interval > 1) return Math.floor(interval) + ' years ago';
		interval = seconds / 2592000;
		if (interval > 1) return Math.floor(interval) + ' months ago';
		interval = seconds / 86400;
		if (interval > 1) return Math.floor(interval) + ' days ago';
		interval = seconds / 3600;
		if (interval > 1) return Math.floor(interval) + ' hours ago';
		interval = seconds / 60;
		if (interval > 1) return Math.floor(interval) + ' minutes ago';
		return Math.floor(seconds) + ' seconds ago';
	}
</script>

<svelte:head>
	<title>{videoQuery.data ? `${videoQuery.data.title} - Aurahub` : 'Watch Video - Aurahub'}</title>
</svelte:head>

<div class="mx-auto grid max-w-[1400px] grid-cols-1 gap-8 py-6 lg:grid-cols-3">
	<!-- Main Content Area -->
	<div class="space-y-6 lg:col-span-2">
		{#if videoQuery.isPending}
			<Skeleton class="aspect-video w-full rounded-xl" />
			<div class="space-y-4">
				<Skeleton class="h-8 w-3/4 rounded" />
				<div class="flex gap-4">
					<Skeleton class="h-12 w-12 rounded-full" />
					<Skeleton class="h-12 w-1/4 rounded" />
				</div>
			</div>
		{:else if videoQuery.isError}
			<div class="bg-muted flex aspect-video w-full items-center justify-center rounded-xl border">
				<p class="font-medium text-destructive">{videoQuery.error.message || 'Failed to load video.'}</p>
				<Button variant="outline" class="mt-4" onclick={() => videoQuery.refetch()}>Try again</Button>
			</div>
		{:else if videoQuery.data}
			<!-- Video Player -->
			<div class="relative aspect-video w-full overflow-hidden rounded-xl border border-border/50 bg-black shadow-lg">
				<iframe
					src={`https://streamtape.com/e/${videoQuery.data.fileId}`}
					class="h-full w-full border-0"
					allowfullscreen
					allow="autoplay"
					title="Video Player"
				></iframe>
			</div>

			<!-- Video Info -->
			<div class="space-y-4">
				<h1 class="text-2xl font-bold tracking-tight">{videoQuery.data.title}</h1>

				<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
					<div class="flex items-center gap-4">
						<a href={`/profile/${videoQuery.data.uploader.username}`} class="shrink-0">
							<img
								src={videoQuery.data.uploader.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${videoQuery.data.uploader.username}`}
								alt={videoQuery.data.uploader.username}
								class="bg-muted h-12 w-12 rounded-full object-cover"
							/>
						</a>
						<div class="flex flex-col">
							<a
								href={`/profile/${videoQuery.data.uploader.username}`}
								class="text-lg font-semibold transition-colors hover:text-primary"
							>
								{videoQuery.data.uploader.username}
							</a>
							<span class="text-muted-foreground text-xs">{uploaderQuery.data?.subscriberCount || 0} subscribers</span>
						</div>
						<Button
							variant={uploaderQuery.data?.isSubscribed ? 'secondary' : 'default'}
							class="ml-2 rounded-full px-5"
							onclick={handleSubscribe}
							disabled={subscribeMutation.isPending}
						>
							{uploaderQuery.data?.isSubscribed ? 'Subscribed' : 'Subscribe'}
						</Button>
					</div>

					<div class="hide-scrollbar flex items-center gap-2 overflow-x-auto pb-2 sm:pb-0">
						<Button
							variant="secondary"
							class={`rounded-full px-4 ${videoQuery.data.isLiked ? 'text-primary' : ''}`}
							onclick={handleLike}
						>
							<ThumbsUp class="mr-2 h-4 w-4 {videoQuery.data.isLiked ? 'fill-primary' : ''}" />
							{videoQuery.data.likesCount}
						</Button>
						<Button variant="secondary" class="rounded-full px-4" onclick={handleShare}>
							<Share2 class="mr-2 h-4 w-4" />
							Share
						</Button>
						<Button variant="secondary" class="rounded-full px-4" onclick={() => (isSaveModalOpen = true)}>
							<ListPlus class="mr-2 h-4 w-4" />
							Save
						</Button>
					</div>
				</div>

				<!-- Description Box -->
				<div class="bg-secondary/50 hover:bg-secondary/70 rounded-xl p-4 text-sm transition-all">
					<div class="mb-1 font-medium">
						{videoQuery.data.views} views • {formatTimeAgo(videoQuery.data.createdAt || new Date().toISOString())}
					</div>
					<div class="text-muted-foreground leading-relaxed whitespace-pre-wrap">
						{videoQuery.data.description || 'No description provided.'}
					</div>
				</div>
			</div>

			<!-- Comments Section -->
			<div class="mt-6 border-t border-border/50 pt-6">
				<h3 class="mb-6 flex items-center gap-2 text-xl font-bold tracking-tight">
					<MessageSquare class="h-5 w-5" />
					{commentsQuery.data?.length || 0} Comments
				</h3>
				<div class="mb-8 flex gap-4">
					<img
						src={userState.user?.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=Guest`}
						alt="Your Avatar"
						class="bg-muted h-10 w-10 shrink-0 rounded-full"
					/>
					<div class="flex-1 space-y-2">
						<Input
							type="text"
							bind:value={newCommentText}
							placeholder={userState.user ? 'Add a comment...' : 'Log in to add a comment...'}
							disabled={!userState.user || commentMutation.isPending}
							class="border-b-only border-muted-foreground/30 bg-transparent px-2 py-1 focus-visible:ring-0 focus-visible:border-primary rounded-none"
						/>
						{#if newCommentText.length > 0}
							<div class="flex justify-end gap-2 mt-2">
								<Button variant="ghost" size="sm" onclick={() => (newCommentText = '')}>Cancel</Button>
								<Button size="sm" onclick={handleCommentSubmit} disabled={commentMutation.isPending}>
									{#if commentMutation.isPending}
										<Loader2 class="mr-2 h-4 w-4 animate-spin" />
									{/if}
									Comment
								</Button>
							</div>
						{/if}
					</div>
				</div>

				<div class="space-y-6">
					{#if commentsQuery.isPending}
						{#each Array(3) as _}
							<div class="flex gap-4 animate-pulse">
								<Skeleton class="h-10 w-10 rounded-full" />
								<div class="flex-1 space-y-2">
									<Skeleton class="h-4 w-1/4 rounded" />
									<Skeleton class="h-4 w-3/4 rounded" />
								</div>
							</div>
						{/each}
					{:else if commentsQuery.data}
						{#each commentsQuery.data as comment}
							<div class="flex gap-4">
								<img
									src={comment.user?.avatar ||
										`https://api.dicebear.com/7.x/identicon/svg?seed=${comment.user?.username || 'user'}`}
									alt="Avatar"
									class="bg-muted h-10 w-10 shrink-0 rounded-full"
								/>
								<div class="space-y-1">
									<div class="flex items-baseline gap-2">
										<span class="text-sm font-semibold">{comment.user?.username || 'Unknown'}</span>
										<span class="text-muted-foreground text-xs"
											>{formatTimeAgo(comment.createdAt || comment.created_at)}</span
										>
									</div>
									<p class="text-sm leading-relaxed">{comment.content}</p>
								</div>
							</div>
						{/each}
					{/if}
				</div>
			</div>

			<SaveToPlaylistModal videoId={videoQuery.data.id} bind:isOpen={isSaveModalOpen} />
		{/if}
	</div>

	<!-- Sidebar Recommendations Area -->
	<div class="flex h-full flex-col space-y-4">
		{#if playlistQuery.isPending && playlistId}
			<Skeleton class="h-64 rounded-xl" />
		{:else if playlistQuery.data && playlistQuery.data.videos}
			<div class="bg-card flex max-h-[500px] flex-col rounded-xl border shadow-sm">
				<div class="bg-secondary/30 border-b p-4">
					<h3 class="line-clamp-1 text-lg font-bold">{playlistQuery.data.title}</h3>
					<p class="text-muted-foreground mt-1 text-xs">
						{playlistQuery.data.owner?.username || 'Playlist'} - {playlistQuery.data.videos.findIndex(
							(v: any) => v.fileId === videoId || v.id === videoId
						) + 1} / {playlistQuery.data.videos.length}
					</p>
				</div>
				<div class="custom-scrollbar flex-1 overflow-y-auto">
					{#each playlistQuery.data.videos as pv, i}
						<a
							href={`/watch/${pv.fileId || pv.id}?list=${playlistId}`}
							class={`hover:bg-muted/50 flex gap-3 p-3 transition-colors ${pv.fileId === videoId || pv.id === videoId ? 'bg-secondary/50 border-l-2 border-primary' : ''}`}
						>
							<div
								class="text-muted-foreground flex w-5 shrink-0 items-center justify-center text-xs"
							>
								{#if pv.fileId === videoId || pv.id === videoId}
									<Play class="h-3.5 w-3.5 fill-primary text-primary" />
								{:else}
									{i + 1}
								{/if}
							</div>
							<VideoThumbnail video={pv} class="h-[72px] w-32 shrink-0 rounded-md" />
							<div class="flex flex-1 flex-col justify-start overflow-hidden">
								<h4
									class={`line-clamp-2 text-sm leading-tight font-semibold ${pv.fileId === videoId || pv.id === videoId ? 'text-primary' : ''}`}
								>
									{pv.title}
								</h4>
								<span class="text-muted-foreground mt-1 text-xs"
									>{pv.uploader?.username || 'Unknown'}</span
								>
							</div>
						</a>
					{/each}
				</div>
			</div>
		{/if}

		<h3 class="mt-2 text-lg font-semibold">Up Next</h3>

		{#if suggestionsQuery.isPending}
			<div class="flex flex-col gap-4">
				{#each Array(6) as _}
					<div class="flex gap-3">
						<Skeleton class="h-24 w-40 rounded-xl" />
						<div class="flex-1 space-y-2 py-1">
							<Skeleton class="h-4 w-full rounded" />
							<Skeleton class="h-4 w-3/4 rounded" />
							<Skeleton class="mt-2 h-3 w-1/2 rounded" />
						</div>
					</div>
				{/each}
			</div>
		{:else if suggestionsQuery.data}
			<div class="flex flex-col gap-3">
				{#each suggestionsQuery.data.pages as page}
					{#each page.videos as nextVideo}
						<a
							href={`/watch/${nextVideo.fileId || nextVideo.id}`}
							class="group hover:bg-muted/50 -mx-2 flex items-start gap-3 rounded-xl p-2 transition-colors"
						>
							<div
								class="bg-muted relative aspect-video w-40 shrink-0 overflow-hidden rounded-lg border"
							>
								<VideoThumbnail video={nextVideo} class="h-full w-full">
									<WatchLaterButton videoId={nextVideo.id} />
								</VideoThumbnail>
							</div>
							<div class="flex flex-1 flex-col py-0.5">
								<h4
									class="line-clamp-2 text-sm leading-tight font-medium transition-colors group-hover:text-primary"
									title={nextVideo.title}
								>
									{nextVideo.title}
								</h4>
								<span class="text-muted-foreground mt-1 text-xs"
									>{nextVideo.uploader?.username || 'Unknown'}</span
								>
								<span class="text-muted-foreground mt-0.5 text-[11px]"
									>{nextVideo.views || 0} views • {formatTimeAgo(
										nextVideo.createdAt || nextVideo.created_at || new Date().toISOString()
									)}</span
								>
							</div>
						</a>
					{/each}
				{/each}

				{#if suggestionsQuery.hasNextPage}
					<div use:infiniteScroll class="flex justify-center py-4">
						{#if suggestionsQuery.isFetchingNextPage}
							<Loader2 class="h-6 w-6 animate-spin text-primary" />
						{/if}
					</div>
				{/if}
			</div>
		{:else}
			<div class="border-muted flex gap-2 rounded-xl border-2 border-dashed p-4 text-center">
				<span class="text-muted-foreground text-sm">No recommendations found.</span>
			</div>
		{/if}
	</div>
</div>

<style>
	.hide-scrollbar::-webkit-scrollbar {
		display: none;
	}
	.hide-scrollbar {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
</style>
