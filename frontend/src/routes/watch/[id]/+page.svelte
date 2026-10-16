<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { contentModeState, initContentMode, setContentMode } from '#lib/consent.svelte';
	import {
		ThumbsUp,
		Share2,
		ListPlus,
		ShieldAlert,
		Hash,
		Tv,
		Minimize2
	} from 'lucide-svelte';
	import SaveToPlaylistModal from '#lib/components/SaveToPlaylistModal.svelte';
	import WatchComments from '#lib/components/WatchComments.svelte';
	import WatchSidebar from '#lib/components/WatchSidebar.svelte';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	import {
		createQuery,
		createMutation,
		useQueryClient
	} from '@tanstack/svelte-query';

	const queryClient = useQueryClient();

	let videoId = $derived(page.params.id);
	let playlistId = $derived(page.url.searchParams.get('list'));
	let isSaveModalOpen = $state(false);
	let isTheaterMode = $state(false);

	function toggleTheaterMode() {
		isTheaterMode = !isTheaterMode;
		try {
			localStorage.setItem('aurahub_theater_mode', String(isTheaterMode));
		} catch {}
	}

	function handleKeyDown(e: KeyboardEvent) {
		const target = e.target as HTMLElement | null;
		if (
			target &&
			(target.tagName === 'INPUT' ||
				target.tagName === 'TEXTAREA' ||
				target.isContentEditable)
		) {
			return;
		}
		if (e.key === 't' || e.key === 'T') {
			e.preventDefault();
			toggleTheaterMode();
		}
	}

	onMount(() => {
		initContentMode();
		try {
			const saved = localStorage.getItem('aurahub_theater_mode');
			if (saved === 'true') isTheaterMode = true;
		} catch {}
		window.addEventListener('keydown', handleKeyDown);
		return () => {
			window.removeEventListener('keydown', handleKeyDown);
		};
	});

	// Queries
	const videoQuery = createQuery(
		() => ({
			queryKey: ['video', videoId],
			queryFn: async () => {
				const data = await fetchApi(`/videos/${videoId}`);
				// Record view async
				fetchApi(`/videos/${data.id || data.fileId || videoId}/view`, { method: 'POST' }).catch(
					console.error
				);
				return data;
			}
		}),
		() => queryClient
	);

	let uploaderUsername = $derived(videoQuery.data?.uploader?.username);
	const uploaderQuery = createQuery(
		() => ({
			queryKey: ['profile', uploaderUsername],
			queryFn: async () => await fetchApi(`/users/${uploaderUsername}/profile`),
			enabled: !!uploaderUsername && videoQuery.data?.uploader?.subscribersCount === undefined
		}),
		() => queryClient
	);

	let subscriberCount = $derived(
		uploaderQuery.data?.subscriberCount ?? videoQuery.data?.uploader?.subscribersCount ?? 0
	);
	let isSubscribed = $derived(
		uploaderQuery.data?.isSubscribed ?? videoQuery.data?.uploader?.isSubscribed ?? false
	);

	let embedUrl = $derived.by(() => {
		const data = videoQuery.data;
		if (!data) return '';
		const fileId =
			data.fileId ||
			data.streamtapeUrl?.match(/streamtape\.com\/(?:v|e)\/([A-Za-z0-9_-]+)/)?.[1];
		if (fileId) {
			return `https://streamtape.com/e/${fileId}/`;
		}
		if (data.streamtapeUrl) {
			return data.streamtapeUrl.replace('streamtape.com/v/', 'streamtape.com/e/');
		}
		return '';
	});

	// Mutations
	const likeMutation = createMutation(
		() => ({
			mutationFn: async (action: 'like' | 'unlike') => {
				const idToUse = videoQuery.data?.id || videoId;
				return await fetchApi(`/videos/${idToUse}/like?action=${action}`, { method: 'POST' });
			},
			onMutate: async (action) => {
				await queryClient.cancelQueries({ queryKey: ['video', videoId] });
				const previous = queryClient.getQueryData(['video', videoId]);
				queryClient.setQueryData(['video', videoId], (old: any) => {
					if (!old) return old;
					const currentLikes = typeof old.likesCount === 'number' ? old.likesCount : old.likes || 0;
					return {
						...old,
						isLiked: action === 'like',
						likesCount: Math.max(0, currentLikes + (action === 'like' ? 1 : -1))
					};
				});
				return { previous };
			},
			onError: (err, newTodo, context: any) => {
				queryClient.setQueryData(['video', videoId], context?.previous);
			},
			onSuccess: (data) => {
				if (data && (typeof data.likesCount === 'number' || typeof data.likes === 'number')) {
					const count = typeof data.likesCount === 'number' ? data.likesCount : data.likes;
					queryClient.setQueryData(['video', videoId], (old: any) => ({
						...old,
						isLiked: data.isLiked,
						likesCount: count
					}));
				}
			},
			onSettled: () => {
				queryClient.invalidateQueries({ queryKey: ['video', videoId] });
			}
		}),
		() => queryClient
	);

	const subscribeMutation = createMutation(
		() => ({
			mutationFn: async () => {
				return await fetchApi(`/users/${uploaderUsername}/subscribe`, { method: 'POST' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['profile', uploaderUsername] });
			}
		}),
		() => queryClient
	);

	// Handlers
	function handleLike() {
		if (!userState.user) return (window.location.href = '/login');
		likeMutation.mutate(videoQuery.data?.isLiked ? 'unlike' : 'like');
	}

	function handleSubscribe() {
		if (!userState.user) return (window.location.href = '/login');
		subscribeMutation.mutate();
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

<div
	class={`mx-auto grid grid-cols-1 gap-6 sm:gap-8 px-4 py-4 sm:py-6 lg:grid-cols-3 transition-all duration-300 ${
		isTheaterMode ? 'max-w-[1920px]' : 'max-w-[1440px]'
	}`}
>
	<!-- Primary Column: Contains Player & Video Details/Comments -->
	<div class={isTheaterMode ? 'contents' : 'space-y-6 lg:col-span-2'}>
		<!-- Video Player Section -->
		<div
			class={`transition-all duration-300 ${
				isTheaterMode ? 'col-span-1 lg:col-span-3 lg:col-start-1 lg:row-start-1' : ''
			}`}
		>
		{#if videoQuery.isPending}
			<Skeleton
				class={`w-full rounded-2xl ${
					isTheaterMode ? 'h-[50vh] sm:h-[65vh] lg:h-[78vh]' : 'aspect-video'
				}`}
			/>
		{:else if videoQuery.isError}
			<div
				class="bg-muted flex aspect-video w-full flex-col items-center justify-center rounded-2xl border p-6 text-center"
			>
				<p class="text-destructive font-medium">
					{videoQuery.error.message || 'Failed to load video.'}
				</p>
				<Button variant="outline" class="mt-4" onclick={() => videoQuery.refetch()}
					>Try again</Button
				>
			</div>
		{:else if videoQuery.data}
			<!-- Video Player with hover theater button -->
			<div
				class={`group relative flex w-full items-center justify-center overflow-hidden rounded-2xl border border-border/50 bg-black shadow-xl transition-all duration-300 ${
					isTheaterMode
						? 'h-[50vh] sm:h-[65vh] lg:h-[78vh] max-h-[860px]'
						: 'aspect-video'
				}`}
			>
				<!-- Quick Hover Theater Mode Toggle Button -->
				<button
					type="button"
					onclick={toggleTheaterMode}
					class="absolute top-3 right-3 z-30 flex h-8 w-8 items-center justify-center rounded-lg bg-black/60 text-white/90 opacity-0 backdrop-blur-md transition-all hover:bg-black/90 hover:scale-105 group-hover:opacity-100"
					title={isTheaterMode ? 'Exit theater mode (T)' : 'Theater mode (T)'}
					aria-label={isTheaterMode ? 'Exit theater mode' : 'Theater mode'}
				>
					{#if isTheaterMode}
						<Minimize2 class="h-4 w-4" />
					{:else}
						<Tv class="h-4 w-4" />
					{/if}
				</button>

				{#if videoQuery.data.isAdult && !contentModeState.isAdult}
					<div class="flex max-w-md flex-col items-center justify-center p-6 text-center">
						<div
							class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl border border-amber-500/30 bg-amber-500/15 text-amber-500"
						>
							<ShieldAlert class="h-8 w-8" />
						</div>
						<div
							class="mb-2 inline-flex items-center gap-1.5 rounded-full bg-amber-500/20 px-3 py-1 text-xs font-semibold text-amber-400"
						>
							18+ Age Restricted Video
						</div>
						<h2 class="mb-2 text-xl font-bold text-white">Mature Content Warning</h2>
						<p class="mb-6 text-xs leading-relaxed text-zinc-300">
							{#if userState.user?.showAdultContent}
								This video contains mature material. Switch to 18+ mode using the header toggle or
								confirm below to watch.
							{:else}
								This video contains mature material. You must enable 18+ content in your profile
								settings before accessing this content.
							{/if}
						</p>
						<div class="flex w-full flex-col justify-center gap-3 sm:flex-row">
							<Button
								variant="outline"
								size="sm"
								class="rounded-full border-zinc-700 bg-zinc-900 text-zinc-200 hover:bg-zinc-800"
								onclick={() => goto('/')}
							>
								Return to Safety
							</Button>
							{#if userState.user?.showAdultContent}
								<Button
									size="sm"
									class="rounded-full bg-amber-600 font-semibold text-white hover:bg-amber-700"
									onclick={() => setContentMode(true)}
								>
									I am 18+ - Watch Video
								</Button>
							{:else}
								<Button
									size="sm"
									class="rounded-full bg-amber-600 font-semibold text-white hover:bg-amber-700"
									onclick={() => goto('/settings')}
								>
									Enable 18+ in Settings
								</Button>
							{/if}
						</div>
					</div>
				{:else if embedUrl}
					<iframe
						src={embedUrl}
						class="h-full w-full border-0"
						allowfullscreen
						allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
						title={videoQuery.data?.title || 'Video player'}
					></iframe>
				{:else}
					<div
						class="flex aspect-video w-full flex-col items-center justify-center p-6 text-center"
					>
						<div
							class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-zinc-900 border border-zinc-800 text-zinc-400"
						>
							<Tv class="h-8 w-8" />
						</div>
						<h3 class="text-lg font-semibold text-zinc-200">Video Processing</h3>
						<p class="mt-1 text-xs text-zinc-400 max-w-sm">
							This video stream is currently being prepared. Please check back in a few moments.
						</p>
					</div>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Video Info & Comments Section -->
	<div
		class={`space-y-6 transition-all duration-300 ${
			isTheaterMode ? 'col-span-1 lg:col-span-2 lg:col-start-1 lg:row-start-2' : ''
		}`}
	>
		{#if videoQuery.isPending}
			<div class="space-y-4">
				<Skeleton class="h-8 w-3/4 rounded" />
				<div class="flex gap-4">
					<Skeleton class="h-12 w-12 rounded-full" />
					<Skeleton class="h-12 w-1/4 rounded" />
				</div>
			</div>
		{:else if videoQuery.data}
			<!-- Video Info -->
			<div class="space-y-4">
				<div class="flex items-start gap-2">
					<h1 class="text-2xl font-bold tracking-tight">{videoQuery.data.title}</h1>
					{#if videoQuery.data.isAdult}
						<span
							class="mt-1 shrink-0 rounded bg-amber-500/20 px-2 py-0.5 text-xs font-bold text-amber-500"
							>18+</span
						>
					{/if}
				</div>

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
							<span class="text-muted-foreground text-xs">{subscriberCount} subscribers</span>
						</div>
						<Button
							variant={isSubscribed ? 'secondary' : 'default'}
							class="ml-2 rounded-full px-5"
							onclick={handleSubscribe}
							disabled={subscribeMutation.isPending}
						>
							{isSubscribed ? 'Subscribed' : 'Subscribe'}
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
						<Button
							variant="secondary"
							class="rounded-full px-4"
							onclick={() => (isSaveModalOpen = true)}
						>
							<ListPlus class="mr-2 h-4 w-4" />
							Save
						</Button>
						<Button
							variant="secondary"
							class={`rounded-full px-4 transition-all ${
								isTheaterMode
									? 'bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm'
									: ''
							}`}
							onclick={toggleTheaterMode}
							title="Toggle theater mode (T)"
						>
							{#if isTheaterMode}
								<Minimize2 class="mr-2 h-4 w-4" />
								Default View
							{:else}
								<Tv class="mr-2 h-4 w-4" />
								Theater Mode
							{/if}
						</Button>
					</div>
				</div>

				<!-- Description Box -->
				<div class="bg-secondary/50 hover:bg-secondary/70 rounded-xl p-4 text-sm transition-all">
					<div class="mb-1 font-medium">
						{videoQuery.data.views} views • {formatTimeAgo(
							videoQuery.data.createdAt || new Date().toISOString()
						)}
					</div>
					<div class="text-muted-foreground leading-relaxed whitespace-pre-wrap">
						{videoQuery.data.description || 'No description provided.'}
					</div>

					{#if videoQuery.data.tags && videoQuery.data.tags.length > 0}
						<div class="mt-3 flex flex-wrap items-center gap-1.5 border-t border-border/40 pt-3">
							{#each videoQuery.data.tags as tag}
								<a
									href={`/search?q=${encodeURIComponent(tag.startsWith('#') ? tag : '#' + tag)}`}
									class="inline-flex items-center gap-1 rounded-full border border-primary/25 bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary transition-all hover:bg-primary hover:text-primary-foreground hover:shadow-sm"
								>
									<Hash class="h-3 w-3" />
									<span>{tag.replace(/^#+/, '')}</span>
								</a>
							{/each}
						</div>
					{/if}
				</div>
			</div>

			<!-- Comments Section Component -->
			<WatchComments
				{videoId}
				resolvedVideoId={videoQuery.data.id}
				{uploaderUsername}
				uploaderId={videoQuery.data.uploaderId || videoQuery.data.uploader?.id}
			/>

			<SaveToPlaylistModal videoId={videoQuery.data.id} bind:isOpen={isSaveModalOpen} />
		{/if}
	</div>
</div>

	<!-- Sidebar Recommendations & Playlist Queue Area -->
	<WatchSidebar
		{videoId}
		{playlistId}
		{isTheaterMode}
	/>
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
