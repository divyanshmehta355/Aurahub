<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { fetchApi, API_URL } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { ThumbsUp, Share2, MessageSquare, Plus, Check, ListPlus } from 'lucide-svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import SaveToPlaylistModal from '#lib/components/SaveToPlaylistModal.svelte';

	import { Play, X, ListVideo } from 'lucide-svelte';

	let videoId = $derived(page.params.id);
	let playlistId = $derived(page.url.searchParams.get('list'));
	
	let video: any = $state(null);
	let isLoading = $state(true);
	let error = $state('');

	let activePlaylist: any = $state(null);
	let isPlaylistLoading = $state(false);

	// Local states for interactivity
	let isLiked = $state(false);
	let likeCount = $state(0);
	let isSubscribed = $state(false);
	let subscriberCount = $state(0);
	let isSaveModalOpen = $state(false);
	
	let comments: any[] = $state([]);
	let newCommentText = $state('');

	let upNextVideos: any[] = $state([]);
	let isSuggestionsLoading = $state(true);
	let suggestionsPage = $state(1);
	let hasMoreSuggestions = $state(true);
	let isLoadingMoreSuggestions = $state(false);

	function infiniteScroll(node: HTMLElement) {
		const observer = new IntersectionObserver((entries) => {
			if (entries[0].isIntersecting) {
				loadMoreSuggestions();
			}
		}, { rootMargin: '200px' });
		
		observer.observe(node);
		
		return {
			destroy() {
				observer.disconnect();
			}
		};
	}

	onMount(() => {
		loadVideo();
	});

	$effect(() => {
		// Reload when videoId changes
		if (videoId) {
			loadVideo();
		}
	});

	$effect(() => {
		if (playlistId) {
			isPlaylistLoading = true;
			fetchApi(`/playlists/${playlistId}`)
				.then(res => {
					const p = res.playlist || res;
					if (res.videos) {
						p.videos = res.videos;
					}
					activePlaylist = p;
				})
				.catch(e => console.error("Failed to fetch playlist", e))
				.finally(() => isPlaylistLoading = false);
		} else {
			activePlaylist = null;
		}
	});

	async function loadVideo() {
		isLoading = true;
		error = '';
		try {
			const data = await fetchApi(`/videos/${videoId}`);
			video = data;
			likeCount = data.likesCount || 0;
			isLiked = data.isLiked || false;

			// Fetch uploader profile for subscription status
			fetchApi(`/users/${data.uploader.username}/profile`)
				.then(profileRes => {
					isSubscribed = profileRes.isSubscribed || false;
					subscriberCount = profileRes.subscriberCount || 0;
				})
				.catch(e => console.error("Failed to fetch uploader profile", e));

			// Record view asynchronously (updates view count and user watch history)
			fetchApi(`/videos/${video.id || video.fileId || videoId}/view`, { method: 'POST' })
				.catch(e => console.error('Failed to record view:', e));
			
			// Load comments
			try {
				const commentsRes = await fetchApi(`/comments/${videoId}`);
				comments = commentsRes.comments || commentsRes.data || [];
			} catch (cerr) {
				console.error("Failed to load comments");
			}

			// Load suggestions
			isSuggestionsLoading = true;
			suggestionsPage = 1;
			hasMoreSuggestions = true;
			try {
				const suggRes = await fetchApi(`/videos/suggestions?exclude=${videoId}&limit=10&page=1`);
				upNextVideos = suggRes.videos || [];
				hasMoreSuggestions = suggRes.currentPage < suggRes.totalPages;
			} catch(serr) {
				console.error("Failed to load suggestions");
			} finally {
				isSuggestionsLoading = false;
			}

			// If we implemented an endpoint to check if user liked/subscribed, we would call it here.
			// For now, we will leave it as false initially.
		} catch (err: any) {
			error = err.message || 'Failed to load video.';
		} finally {
			isLoading = false;
		}
	}

	async function handleCommentSubmit() {
		if (!newCommentText.trim() || !userState.user) return;
		try {
			const res = await fetchApi('/comments', {
				method: 'POST',
				body: JSON.stringify({ videoId: video.id || video.fileId, content: newCommentText.trim() })
			});
			if (res.comment) {
				comments = [res.comment, ...comments];
			} else {
				// reload comments
				const commentsRes = await fetchApi(`/comments/${videoId}`);
				comments = commentsRes.comments || commentsRes.data || [];
			}
			newCommentText = '';
		} catch (err) {
			alert('Failed to post comment');
		}
	}

	async function handleLike() {
		if (!userState.user) {
			window.location.href = '/login';
			return;
		}

		const action = isLiked ? 'unlike' : 'like';
		try {
			await fetchApi(`/videos/${video.id}/like?action=${action}`, { method: 'POST' });
			isLiked = !isLiked;
			likeCount += isLiked ? 1 : -1;
		} catch (err) {
			console.error("Failed to toggle like", err);
		}
	}

	async function handleSubscribe() {
		if (!userState.user) {
			window.location.href = '/login';
			return;
		}
		
		try {
			const res = await fetchApi(`/users/${video.uploader.username}/subscribe`, { method: 'POST' });
			isSubscribed = res.isSubscribed;
			subscriberCount = res.subscriberCount;
		} catch (err) {
			console.error("Failed to toggle subscribe", err);
		}
	}

	async function loadMoreSuggestions() {
		if (isLoadingMoreSuggestions || !hasMoreSuggestions) return;
		isLoadingMoreSuggestions = true;
		suggestionsPage++;
		try {
			const suggRes = await fetchApi(`/videos/suggestions?exclude=${videoId}&limit=10&page=${suggestionsPage}`);
			if (suggRes.videos && suggRes.videos.length > 0) {
				upNextVideos = [...upNextVideos, ...suggRes.videos];
				hasMoreSuggestions = suggRes.currentPage < suggRes.totalPages;
			} else {
				hasMoreSuggestions = false;
			}
		} catch(serr) {
			console.error("Failed to load more suggestions");
		} finally {
			isLoadingMoreSuggestions = false;
		}
	}

	function handleShare() {
		navigator.clipboard.writeText(window.location.href);
		alert("Link copied to clipboard!");
	}

	function formatTimeAgo(dateString: string) {
		const date = new Date(dateString);
		const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
		let interval = seconds / 31536000;
		if (interval > 1) return Math.floor(interval) + " years ago";
		interval = seconds / 2592000;
		if (interval > 1) return Math.floor(interval) + " months ago";
		interval = seconds / 86400;
		if (interval > 1) return Math.floor(interval) + " days ago";
		interval = seconds / 3600;
		if (interval > 1) return Math.floor(interval) + " hours ago";
		interval = seconds / 60;
		if (interval > 1) return Math.floor(interval) + " minutes ago";
		return Math.floor(seconds) + " seconds ago";
	}
</script>

<svelte:head>
	<title>{video ? `${video.title} - Aurahub` : 'Watch Video - Aurahub'}</title>
</svelte:head>

<div class="max-w-[1400px] mx-auto py-6 grid grid-cols-1 lg:grid-cols-3 gap-8">
	<!-- Main Content Area -->
	<div class="lg:col-span-2 space-y-6">
		{#if isLoading}
			<div class="w-full aspect-video bg-muted animate-pulse rounded-xl"></div>
			<div class="space-y-4 animate-pulse">
				<div class="h-8 bg-muted rounded w-3/4"></div>
				<div class="flex gap-4">
					<div class="h-12 w-12 bg-muted rounded-full shrink-0"></div>
					<div class="h-12 bg-muted rounded w-1/4"></div>
				</div>
			</div>
		{:else if error}
			<div class="w-full aspect-video bg-muted rounded-xl flex items-center justify-center border">
				<p class="text-red-500 font-medium">{error}</p>
			</div>
		{:else if video}
			<!-- Video Player -->
			<div class="w-full aspect-video bg-black rounded-xl overflow-hidden shadow-lg border border-border/50 relative">
				<iframe 
					src={`https://streamtape.com/e/${video.fileId}`} 
					class="w-full h-full border-0"
					allowfullscreen 
					allow="autoplay"
					title="Video Player"
				></iframe>
			</div>

			<!-- Video Info -->
			<div class="space-y-4">
				<h1 class="text-2xl font-bold tracking-tight">{video.title}</h1>
				
				<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
					<div class="flex items-center gap-4">
						<a href={`/channel/${video.uploader.username}`} class="shrink-0">
							<img 
								src={video.uploader.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader.username}`} 
								alt={video.uploader.username} 
								class="h-12 w-12 rounded-full object-cover bg-muted"
							/>
						</a>
						<div class="flex flex-col">
							<a href={`/channel/${video.uploader.username}`} class="font-semibold text-lg hover:text-primary transition-colors">
								{video.uploader.username}
							</a>
							<span class="text-xs text-muted-foreground">{subscriberCount} subscribers</span>
						</div>
						<button 
							class="ml-2 inline-flex h-9 items-center justify-center rounded-full px-5 text-sm font-medium transition-colors {isSubscribed ? 'bg-secondary text-secondary-foreground hover:bg-secondary/80' : 'bg-primary text-primary-foreground hover:bg-primary/90'}"
							onclick={handleSubscribe}
						>
							{isSubscribed ? 'Subscribed' : 'Subscribe'}
						</button>
					</div>

					<div class="flex items-center gap-2 overflow-x-auto pb-2 sm:pb-0 hide-scrollbar">
						<button 
							class="inline-flex h-9 items-center justify-center rounded-full bg-secondary text-secondary-foreground hover:bg-secondary/80 px-4 text-sm font-medium transition-colors shrink-0 {isLiked ? 'text-primary' : ''}"
							onclick={handleLike}
						>
							<ThumbsUp class="mr-2 h-4 w-4 {isLiked ? 'fill-primary text-primary' : ''}" />
							{likeCount}
						</button>
						<button 
							class="inline-flex h-9 items-center justify-center rounded-full bg-secondary text-secondary-foreground hover:bg-secondary/80 px-4 text-sm font-medium transition-colors shrink-0"
							onclick={handleShare}
						>
							<Share2 class="mr-2 h-4 w-4" />
							Share
						</button>
						<button 
							class="inline-flex h-9 items-center justify-center rounded-full bg-secondary text-secondary-foreground hover:bg-secondary/80 px-4 text-sm font-medium transition-colors shrink-0"
							onclick={() => isSaveModalOpen = true}
						>
							<ListPlus class="mr-2 h-4 w-4" />
							Save
						</button>
					</div>
				</div>

				<!-- Description Box -->
				<div class="bg-secondary/50 rounded-xl p-4 text-sm transition-all hover:bg-secondary/70">
					<div class="font-medium mb-1">
						{video.views} views • {formatTimeAgo(video.createdAt || new Date().toISOString())}
					</div>
					<div class="whitespace-pre-wrap text-muted-foreground leading-relaxed">
						{video.description || "No description provided."}
					</div>
				</div>
			</div>

			<!-- Comments Section -->
			<div class="pt-6 mt-6 border-t border-border/50">
				<h3 class="text-xl font-bold tracking-tight mb-6 flex items-center gap-2">
					<MessageSquare class="h-5 w-5" />
					{comments.length} Comments
				</h3>
				<div class="flex gap-4 mb-8">
					<img 
						src={userState.user?.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=Guest`} 
						alt="Your Avatar" 
						class="h-10 w-10 rounded-full shrink-0 bg-muted" 
					/>
					<div class="flex-1 space-y-2">
						<input 
							type="text" 
							bind:value={newCommentText}
							placeholder={userState.user ? "Add a comment..." : "Log in to add a comment..."} 
							disabled={!userState.user}
							class="w-full bg-transparent border-b border-muted-foreground/30 px-2 py-1 text-sm focus:outline-none focus:border-primary transition-colors disabled:opacity-50"
						/>
						{#if newCommentText.length > 0}
							<div class="flex justify-end gap-2">
								<button onclick={() => newCommentText = ''} class="text-xs font-medium px-3 py-1.5 rounded-full hover:bg-secondary transition-colors">Cancel</button>
								<button onclick={handleCommentSubmit} class="text-xs font-medium px-3 py-1.5 rounded-full bg-primary text-primary-foreground">Comment</button>
							</div>
						{/if}
					</div>
				</div>

				<div class="space-y-6">
					{#each comments as comment}
						<div class="flex gap-4">
							<img 
								src={comment.user?.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${comment.user?.username || 'user'}`} 
								alt="Avatar" 
								class="h-10 w-10 rounded-full shrink-0 bg-muted" 
							/>
							<div class="space-y-1">
								<div class="flex items-baseline gap-2">
									<span class="font-semibold text-sm">{comment.user?.username || 'Unknown'}</span>
									<span class="text-xs text-muted-foreground">{formatTimeAgo(comment.createdAt || comment.created_at)}</span>
								</div>
								<p class="text-sm leading-relaxed">{comment.content}</p>
							</div>
						</div>
					{/each}
				</div>
			</div>
			
			<SaveToPlaylistModal videoId={video.id} bind:isOpen={isSaveModalOpen} />
		{/if}
	</div>

	<!-- Sidebar Recommendations Area -->
	<div class="space-y-4 flex-col flex h-full">
		
		{#if isPlaylistLoading}
			<div class="border rounded-xl p-4 bg-muted/20 animate-pulse h-64"></div>
		{:else if activePlaylist && activePlaylist.videos}
			<div class="border rounded-xl bg-card shadow-sm flex flex-col max-h-[500px]">
				<div class="p-4 border-b bg-secondary/30">
					<h3 class="font-bold text-lg line-clamp-1">{activePlaylist.title}</h3>
					<p class="text-xs text-muted-foreground mt-1">
						{activePlaylist.owner?.username || 'Playlist'} - {activePlaylist.videos.findIndex((v: any) => v.fileId === videoId || v.id === videoId) + 1} / {activePlaylist.videos.length}
					</p>
				</div>
				<div class="overflow-y-auto flex-1 custom-scrollbar">
					{#each activePlaylist.videos as pv, i}
						<a 
							href={`/watch/${pv.fileId || pv.id}?list=${playlistId}`}
							class={`flex gap-3 p-3 transition-colors hover:bg-muted/50 ${(pv.fileId === videoId || pv.id === videoId) ? 'bg-secondary/50 border-l-2 border-primary' : ''}`}
						>
							<div class="w-5 flex items-center justify-center shrink-0 text-xs text-muted-foreground">
								{#if pv.fileId === videoId || pv.id === videoId}
									<Play class="w-3.5 h-3.5 text-primary fill-primary" />
								{:else}
									{i + 1}
								{/if}
							</div>
							<VideoThumbnail 
								video={pv} 
								class="w-32 h-[72px] shrink-0 rounded-md" 
							/>
							<div class="flex-1 flex flex-col justify-start overflow-hidden">
								<h4 class={`text-sm font-semibold line-clamp-2 leading-tight ${(pv.fileId === videoId || pv.id === videoId) ? 'text-primary' : ''}`}>
									{pv.title}
								</h4>
								<span class="text-xs text-muted-foreground mt-1">{pv.uploader?.username || 'Unknown'}</span>
							</div>
						</a>
					{/each}
				</div>
			</div>
		{/if}

		<h3 class="font-semibold text-lg mt-2">Up Next</h3>
		
		{#if isSuggestionsLoading}
			<div class="flex flex-col gap-4">
				{#each Array(6) as _}
					<div class="flex gap-3 animate-pulse">
						<div class="w-40 h-24 bg-muted rounded-xl shrink-0"></div>
						<div class="flex-1 space-y-2 py-1">
							<div class="h-4 bg-muted rounded w-full"></div>
							<div class="h-4 bg-muted rounded w-3/4"></div>
							<div class="h-3 bg-muted rounded w-1/2 mt-2"></div>
						</div>
					</div>
				{/each}
			</div>
		{:else if upNextVideos.length > 0}
			<div class="flex flex-col gap-3">
				{#each upNextVideos as nextVideo}
					<a href={`/watch/${nextVideo.fileId || nextVideo.id}`} class="flex gap-3 group items-start hover:bg-muted/50 p-2 -mx-2 rounded-xl transition-colors">
						<div class="w-40 shrink-0 aspect-video rounded-lg overflow-hidden relative border bg-muted">
							<VideoThumbnail video={nextVideo} class="w-full h-full">
								<WatchLaterButton videoId={nextVideo.id} />
							</VideoThumbnail>
						</div>
						<div class="flex flex-col flex-1 py-0.5">
							<h4 class="font-medium text-sm line-clamp-2 leading-tight group-hover:text-primary transition-colors" title={nextVideo.title}>{nextVideo.title}</h4>
							<span class="text-xs text-muted-foreground mt-1">{nextVideo.uploader?.username || 'Unknown'}</span>
							<span class="text-[11px] text-muted-foreground mt-0.5">{nextVideo.views || 0} views • {formatTimeAgo(nextVideo.createdAt || nextVideo.created_at || new Date().toISOString())}</span>
						</div>
					</a>
				{/each}

				{#if hasMoreSuggestions}
					<div use:infiniteScroll class="flex justify-center py-4">
						{#if isLoadingMoreSuggestions}
							<div class="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
						{/if}
					</div>
				{/if}
			</div>
		{:else}
			<div class="flex gap-2 p-4 text-center border-2 border-dashed border-muted rounded-xl">
				<span class="text-sm text-muted-foreground">No recommendations found.</span>
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
