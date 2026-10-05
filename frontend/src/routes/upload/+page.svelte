<script lang="ts">
	import { fetchApi } from '#lib/api';
	import {
		UploadCloud,
		FileVideo,
		X,
		CheckCircle,
		Link as LinkIcon,
		ListVideo,
		Loader2
	} from 'lucide-svelte';
	import PlaylistUploadForm from '#lib/components/PlaylistUploadForm.svelte';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	import { Label } from '#lib/components/ui/label';
	import { Textarea } from '#lib/components/ui/textarea';
	import * as Card from '#lib/components/ui/card';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';

	const queryClient = useQueryClient();

	let uploadType = $state('direct'); // 'direct', 'remote', 'playlist'

	// Fetch playlists for PlaylistBatch mode if needed
	const playlistsQuery = createQuery(() => ({
		queryKey: ['my-playlists'],
		queryFn: async () => {
			const res = await fetchApi('/playlists/my-playlists');
			return res || [];
		}
	}), () => queryClient);

	// Common form
	let title = $state('');
	let description = $state('');
	let category = $state('Other');
	let isShort = $state(false);

	// Direct
	let file: File | null = $state(null);

	// Remote
	let videoUrl = $state('');

	let isUploading = $state(false);
	let uploadProgress = $state(0);
	let error = $state('');
	let success = $state(false);
	let fileId = $state('');
	let statusMessage = $state('');

	const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB

	function handleFileSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			file = target.files[0];
			title = file.name.replace(/\.[^/.]+$/, '');
			error = '';
		}
	}

	async function uploadDirect(e: Event) {
		e.preventDefault();
		if (!file) {
			error = 'Please select a video file.';
			return;
		}

		isUploading = true;
		error = '';
		uploadProgress = 0;
		statusMessage = 'Uploading chunks...';

		try {
			const initRes = await fetchApi('/videos/get-upload-url');
			const serverFileId = initRes.fileId;
			const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

			let finalVideoId = '';
			for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
				const start = chunkIndex * CHUNK_SIZE;
				const end = Math.min(start + CHUNK_SIZE, file.size);
				const chunk = file.slice(start, end);

				const formData = new FormData();
				formData.append('uploadId', serverFileId);
				formData.append('chunkIndex', chunkIndex.toString());
				formData.append('totalChunks', totalChunks.toString());
				formData.append('fileName', file.name);
				formData.append('chunk', chunk);

				const res = await fetchApi('/videos/upload-chunk', { method: 'POST', body: formData });
				if (res && res.completed) {
					finalVideoId = res.videoId;
				}
				uploadProgress = Math.round(((chunkIndex + 1) / totalChunks) * 100);
			}

			statusMessage = 'Creating record...';
			const finalData = new FormData();
			finalData.append('videoId', finalVideoId);
			finalData.append('title', title);
			finalData.append('description', description);
			finalData.append('category', category);
			finalData.append('visibility', 'public');
			finalData.append('isShort', isShort.toString());

			await fetchApi('/videos/create-record', {
				method: 'POST',
				body: finalData
			});

			fileId = serverFileId;
			success = true;
		} catch (err: any) {
			error = err.message || 'Upload failed.';
		} finally {
			isUploading = false;
		}
	}

	async function uploadRemote(e: Event) {
		e.preventDefault();
		if (!videoUrl) {
			error = 'Please enter a video URL.';
			return;
		}

		isUploading = true;
		error = '';
		uploadProgress = 0;
		statusMessage = 'Queuing remote upload...';

		try {
			const startRes = await fetchApi('/videos/remote-upload/start', {
				method: 'POST',
				body: JSON.stringify({ videoUrl })
			});
			const remoteId = startRes.id;
			if (!remoteId) throw new Error('Failed to get remote upload ID.');

			statusMessage = 'Waiting in queue...';

			// Polling
			while (true) {
				await new Promise((r) => setTimeout(r, 4000));
				const statusRes = await fetchApi(`/videos/remote-upload/status?id=${remoteId}`);
				const statusData = statusRes[remoteId];

				if (statusData) {
					if (statusData.status === 'finished') {
						uploadProgress = 100;
						statusMessage = 'Publishing video...';
						const finalVideoId = statusData.linkid;

						const finalData = new FormData();
						finalData.append('videoId', finalVideoId);
						finalData.append('title', title);
						finalData.append('description', description);
						finalData.append('category', category);
						finalData.append('visibility', 'public');
						finalData.append('isShort', isShort.toString());

						await fetchApi('/videos/create-record', {
							method: 'POST',
							body: finalData
						});

						fileId = finalVideoId;
						success = true;
						break;
					} else if (statusData.status === 'error') {
						throw new Error(statusData.error_message || 'Remote download error.');
					} else {
						const loaded = statusData.bytes_loaded || 0;
						const total = statusData.bytes_total || 0;
						if (total > 0) {
							uploadProgress = Math.floor((loaded / total) * 100);
							statusMessage = `Downloading... ${uploadProgress}%`;
						}
					}
				}
			}
		} catch (err: any) {
			error = err.message || 'Remote upload failed.';
		} finally {
			isUploading = false;
		}
	}

	function reset() {
		file = null;
		title = '';
		description = '';
		category = 'Other';
		isShort = false;
		videoUrl = '';
		isUploading = false;
		uploadProgress = 0;
		error = '';
		success = false;
		fileId = '';
		statusMessage = '';
	}
</script>

<svelte:head>
	<title>Upload Video - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-4xl px-4 py-8 sm:px-6">
	<Card.Root class="overflow-hidden shadow-sm">
		<div class="bg-muted/20 border-b p-6 text-center">
			<h1 class="text-2xl font-bold tracking-tight">Upload Video</h1>
			<p class="text-muted-foreground mt-1 text-sm">
				Share your content with the Aurahub community.
			</p>
		</div>

		<div class="bg-muted/5 flex justify-center border-b p-4 sm:p-6">
			<div class="bg-muted flex w-full max-w-2xl rounded-xl p-1">
				<button
					onclick={() => (uploadType = 'direct')}
					class={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-all ${uploadType === 'direct' ? 'text-foreground bg-background shadow' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<UploadCloud class="h-4 w-4" /> Direct Upload
				</button>
				<button
					onclick={() => (uploadType = 'remote')}
					class={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-all ${uploadType === 'remote' ? 'text-foreground bg-background shadow' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<LinkIcon class="h-4 w-4" /> Remote URL
				</button>
				<button
					onclick={() => (uploadType = 'playlist')}
					class={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-sm font-semibold transition-all ${uploadType === 'playlist' ? 'text-foreground bg-background shadow' : 'text-muted-foreground hover:text-foreground'}`}
				>
					<ListVideo class="h-4 w-4" /> Playlist Batch
				</button>
			</div>
		</div>

		<Card.Content class="p-6 sm:p-8">
			{#if success}
				<div
					class="animate-in zoom-in flex flex-col items-center justify-center py-12 text-center duration-300"
				>
					<div
						class="mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-green-100 text-green-600"
					>
						<CheckCircle class="h-10 w-10" />
					</div>
					<h2 class="mb-2 text-2xl font-bold">Upload Complete!</h2>
					<p class="text-muted-foreground mb-8 max-w-md">
						Your video has been successfully uploaded and is now processing.
					</p>
					<div class="flex gap-4">
						<Button variant="outline" class="rounded-full px-6" onclick={reset}>
							Upload Another
						</Button>
						<a href={`/watch/${fileId}`}>
							<Button class="rounded-full px-6">View Video</Button>
						</a>
					</div>
				</div>
			{:else if uploadType === 'playlist'}
				<PlaylistUploadForm playlists={playlistsQuery.data || []} />
			{:else}
				<form
					onsubmit={uploadType === 'direct' ? uploadDirect : uploadRemote}
					class="mx-auto max-w-2xl space-y-8"
				>
					{#if error}
						<div
							class="flex items-center justify-between rounded-lg bg-destructive/10 p-4 text-sm font-medium text-destructive"
						>
							{error}
							<button type="button" onclick={() => (error = '')}><X class="h-4 w-4" /></button>
						</div>
					{/if}

					<div class="space-y-4">
						<div class="space-y-1.5">
							<Label for="title">Title <span class="text-destructive">*</span></Label>
							<Input
								type="text"
								id="title"
								bind:value={title}
								required
								disabled={isUploading}
								placeholder="Video Title"
							/>
						</div>

						{#if uploadType === 'remote'}
							<div class="space-y-1.5">
								<Label for="url">Remote URL (Direct MP4 / Stream URL) <span class="text-destructive">*</span></Label>
								<Input
									type="url"
									id="url"
									bind:value={videoUrl}
									required
									disabled={isUploading}
									placeholder="https://example.com/video.mp4"
									class="font-mono"
								/>
							</div>
						{:else}
							<div class="space-y-1.5">
								<Label>Video File <span class="text-destructive">*</span></Label>
								<div class="relative">
									<input
										type="file"
										id="videoFile"
										accept="video/mp4,video/webm,video/ogg"
										class="hidden"
										onchange={handleFileSelect}
										disabled={isUploading}
									/>
									<label
										for="videoFile"
										class="hover:bg-muted/50 border-muted-foreground/30 flex h-40 w-full cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed bg-background/50 transition-colors"
									>
										<div class="flex flex-col items-center justify-center px-4 pt-5 pb-6 text-center">
											{#if file}
												<FileVideo class="mb-3 h-10 w-10 text-primary" />
												<p class="text-sm font-semibold">{file.name}</p>
												<p class="text-muted-foreground mt-1 text-xs">
													{(file.size / (1024 * 1024)).toFixed(2)} MB
												</p>
											{:else}
												<UploadCloud class="text-muted-foreground mb-3 h-10 w-10 opacity-50" />
												<p class="mb-1 text-sm font-semibold">Click to select video</p>
												<p class="text-muted-foreground text-xs">MP4, WEBM, or OGG</p>
											{/if}
										</div>
									</label>
								</div>
							</div>
						{/if}

						<div class="grid grid-cols-2 gap-4">
							<div class="space-y-1.5">
								<Label for="category">Category</Label>
								<select
									id="category"
									bind:value={category}
									disabled={isUploading}
									class="border-input h-10 w-full rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-primary focus:outline-none disabled:opacity-50"
								>
									<option value="Other">Other</option>
									<option value="Gaming">Gaming</option>
									<option value="Education">Education</option>
									<option value="Entertainment">Entertainment</option>
									<option value="Music">Music</option>
									<option value="Science">Science & Tech</option>
								</select>
							</div>
						</div>

						<div class="space-y-1.5">
							<Label for="description">Description</Label>
							<Textarea
								id="description"
								bind:value={description}
								disabled={isUploading}
								rows={4}
								placeholder="Tell viewers about your video"
							/>
						</div>

						<div class="flex items-center space-x-2">
							<input
								type="checkbox"
								id="isShort"
								bind:checked={isShort}
								disabled={isUploading}
								class="border-input h-4 w-4 rounded text-primary focus:ring-primary disabled:opacity-50"
							/>
							<Label for="isShort" class="cursor-pointer">Upload as Short (Vertical Video)</Label>
						</div>
					</div>

					{#if isUploading}
						<div class="bg-muted/30 space-y-2 rounded-xl border p-4">
							<div class="flex justify-between text-sm font-medium">
								<span>{statusMessage}</span>
								<span>{uploadProgress}%</span>
							</div>
							<div class="bg-secondary h-2 w-full overflow-hidden rounded-full">
								<div
									class="h-2 bg-primary transition-all duration-300"
									style="width: {uploadProgress}%"
								></div>
							</div>
						</div>
					{/if}

					<div class="pt-4">
						<Button
							type="submit"
							disabled={isUploading || (uploadType === 'direct' && !file)}
							class="w-full h-12 text-md"
						>
							{#if isUploading}
								<Loader2 class="mr-2 h-5 w-5 animate-spin" />
								Processing...
							{:else}
								Upload Video
							{/if}
						</Button>
					</div>
				</form>
			{/if}
		</Card.Content>
	</Card.Root>
</div>
