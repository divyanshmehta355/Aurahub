<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { Image as ImageIcon } from 'lucide-svelte';

	let username = $derived(page.params.username);
	let profile: any = $state(null);
	let isLoading = $state(true);
	let error = $state('');

	let isOwnProfile = $derived(userState.user?.id === profile?.user?.id);

	async function loadProfile() {
		isLoading = true;
		error = '';
		try {
			// Backend expects identifier, which we send as the username
			const data = await fetchApi(`/users/profile/${username}`);
			profile = data;
		} catch (err: any) {
			error = err.message || 'User not found.';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (username) {
			loadProfile();
		}
	});

	async function handleSubscribe() {
		if (!userState.user) {
			alert('Please log in to subscribe.');
			return;
		}

		const originalProfile = { ...profile };
		
		// Optimistic update
		profile.user.isSubscribed = !profile.user.isSubscribed;
		profile.user.subscriberCount += profile.user.isSubscribed ? 1 : -1;

		try {
			await fetchApi(`/interactions/subscribe/${profile.user.id}`, {
				method: 'POST'
			});
		} catch (err) {
			alert('An error occurred. Please try again.');
			// Revert
			profile = originalProfile;
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
	<title>{username ? `${username} - Aurahub` : 'Profile - Aurahub'}</title>
</svelte:head>

{#if isLoading}
	<main class="animate-pulse pb-12">
		<div class="bg-muted mb-8 h-48 w-full md:h-64"></div>
		<div class="container mx-auto px-6">
			<div class="mb-8 -mt-20 flex items-end gap-6">
				<div class="bg-muted ring-background h-32 w-32 rounded-full ring-4"></div>
				<div class="mb-2 h-10 w-48 rounded bg-gray-300 dark:bg-gray-700"></div>
			</div>
			<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
				{#each Array(4) as _}
					<div class="flex flex-col space-y-3">
						<div class="bg-muted aspect-video w-full rounded-xl"></div>
						<div class="flex gap-3">
							<div class="bg-muted h-10 w-10 shrink-0 rounded-full"></div>
							<div class="w-full space-y-2">
								<div class="bg-muted h-4 w-3/4 rounded"></div>
								<div class="bg-muted h-3 w-1/2 rounded"></div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</main>
{:else if error || !profile}
	<div class="p-12 text-center font-semibold text-rose-500">{error}</div>
{:else}
	<main class="pb-12 fade-in">
		<!-- CHANNEL BANNER -->
		<div class="bg-muted relative h-48 w-full md:h-72">
			{#if profile.user.banner}
				<img
					src={profile.user.banner}
					alt={`${profile.user.username}'s banner`}
					class="h-full w-full object-cover"
				/>
			{:else}
				<div class="flex h-full w-full flex-col items-center justify-center opacity-40">
					<ImageIcon size={64} class="text-muted-foreground mb-4" />
					<span class="text-muted-foreground font-medium">No banner provided</span>
				</div>
			{/if}
		</div>

		<div class="container mx-auto px-4 sm:px-6">
			<!-- PROFILE HEADER INFO -->
			<div
				class="bg-card border-border relative z-10 mb-10 -mt-12 flex flex-col gap-6 rounded-3xl border p-6 shadow-sm sm:-mt-20 md:flex-row md:items-end md:justify-between"
			>
				<div class="flex w-full flex-col gap-6 md:flex-row md:items-end">
					<div class="shrink-0">
						<img
							src={profile.user.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${profile.user.username}`}
							alt={profile.user.username}
							class="bg-card ring-background h-24 w-24 rounded-full object-cover ring-4 sm:h-36 sm:w-36"
						/>
					</div>

					<div class="grow pb-2">
						<h1 class="text-foreground tracking-tight font-display text-3xl font-extrabold sm:text-4xl">
							{profile.user.username}
						</h1>
						<p class="text-muted-foreground mt-2 flex items-center gap-2 font-medium">
							<span class="text-foreground font-bold">{profile.user.subscriberCount || 0}</span> subscribers
							<span>•</span>
							<span class="text-foreground font-bold">{profile.videos?.length || 0}</span> videos
						</p>

						{#if profile.user.bio}
							<p class="text-muted-foreground mt-4 max-w-2xl whitespace-pre-line text-sm sm:text-base">
								{profile.user.bio}
							</p>
						{/if}
					</div>

					<div class="shrink-0 pb-2">
						{#if isOwnProfile}
							<a
								href="/profile"
								class="bg-muted text-foreground hover:bg-muted block rounded-xl px-8 py-3 text-center font-semibold shadow-sm transition-colors"
							>
								Customize Channel
							</a>
						{:else if userState.user}
							<button
								onclick={handleSubscribe}
								class={`w-full rounded-xl px-8 py-3 font-semibold shadow-sm transition-all md:w-auto ${
									profile.user.isSubscribed
										? 'bg-muted text-foreground hover:bg-muted'
										: 'bg-foreground text-background hover:bg-foreground/90 hover:-translate-y-0.5 hover:shadow-md transform'
								}`}
							>
								{profile.user.isSubscribed ? 'Subscribed' : 'Subscribe'}
							</button>
						{/if}
					</div>
				</div>
			</div>

			<!-- UPLOADS SECTION -->
			<h2 class="text-foreground tracking-tight font-display mb-6 text-2xl font-bold">Uploads</h2>

			<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
				{#if profile.videos && profile.videos.length > 0}
					{#each profile.videos as video}
						<a href={`/watch/${video.fileId || video.id}`} class="group flex flex-col space-y-3">
							<!-- Thumbnail -->
							<VideoThumbnail
								{video}
								class="aspect-video rounded-xl"
								imgClass="transition-transform duration-300 group-hover:scale-105"
							>
								<div class="absolute right-2 bottom-2 rounded bg-black/80 px-2 py-1 text-xs font-medium text-white backdrop-blur-sm">
									{formatDuration(video.duration)}
								</div>
							</VideoThumbnail>

							<!-- Metadata -->
							<div class="flex gap-3">
								<div class="flex flex-col overflow-hidden">
									<h3 class="line-clamp-2 text-sm leading-tight font-semibold transition-colors group-hover:text-primary">
										{video.title}
									</h3>
									<div class="text-muted-foreground mt-1 space-y-0.5 text-xs">
										<div class="flex items-center gap-1">
											<span>{video.views || 0} views</span>
											<span class="text-[10px]">•</span>
											<span>{new Date(video.created_at).toLocaleDateString()}</span>
										</div>
									</div>
								</div>
							</div>
						</a>
					{/each}
				{:else}
					<div class="border-border bg-card col-span-full rounded-2xl border p-16 text-center shadow-sm">
						<p class="text-muted-foreground text-lg font-medium">This creator hasn't uploaded any videos yet.</p>
						<p class="text-muted-foreground mt-2 text-sm">Check back later for new content!</p>
					</div>
				{/if}
			</div>
		</div>
	</main>
{/if}

<style>
	.fade-in {
		animation: fadeIn 0.4s ease-in-out;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
			transform: translateY(20px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
</style>
