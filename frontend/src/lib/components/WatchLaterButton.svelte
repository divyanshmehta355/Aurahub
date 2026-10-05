<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { Clock, Check } from 'lucide-svelte';

	let { videoId } = $props();
	let isSaved = $state(false);

	async function toggleWatchLater(e: Event) {
		e.preventDefault();
		e.stopPropagation();

		if (!userState.user) {
			window.location.href = '/login';
			return;
		}

		try {
			const action = isSaved ? 'remove' : 'add';
			await fetchApi('/user/watch-later', {
				method: 'POST',
				body: JSON.stringify({ videoId, action })
			});
			isSaved = !isSaved;
		} catch (err) {
			console.error('Failed to toggle watch later', err);
		}
	}
</script>

{#if userState.user}
	<button
		onclick={toggleWatchLater}
		title={isSaved ? 'Remove from Watch Later' : 'Watch Later'}
		class="absolute top-2 right-2 z-20 flex h-8 w-8 items-center justify-center rounded-full bg-black/60 text-white opacity-0 shadow-lg backdrop-blur-md transition-all duration-300 group-hover:opacity-100 hover:scale-110 hover:bg-black/80"
	>
		{#if isSaved}
			<Check class="h-4 w-4 text-emerald-400" />
		{:else}
			<Clock class="h-4 w-4" />
		{/if}
	</button>
{/if}
