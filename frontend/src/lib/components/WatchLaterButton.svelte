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
			console.error("Failed to toggle watch later", err);
		}
	}
</script>

{#if userState.user}
	<button
		onclick={toggleWatchLater}
		title={isSaved ? "Remove from Watch Later" : "Watch Later"}
		class="absolute top-2 right-2 h-8 w-8 flex items-center justify-center bg-black/60 hover:bg-black/80 backdrop-blur-md text-white rounded-full transition-all duration-300 opacity-0 group-hover:opacity-100 z-20 hover:scale-110 shadow-lg"
	>
		{#if isSaved}
			<Check class="h-4 w-4 text-emerald-400" />
		{:else}
			<Clock class="h-4 w-4" />
		{/if}
	</button>
{/if}
