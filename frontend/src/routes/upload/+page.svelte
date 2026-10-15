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
	const playlistsQuery = createQuery(
		() => ({
			queryKey: ['my-playlists'],
			queryFn: async () => {
				const res = await fetchApi('/playlists/my-playlists');
				return res || [];
			}
		}),
		() => queryClient
	);

	// Common form
	let title = $state('');
	let description = $state('');
	let category = $state('Other');
	let isShort = $state(false);
	let isAdult = $state(false);

	// Direct
	let file: File | null = $state(null);

	// Remote
	let videoUrl = $state('');

	let isUploading = $state(false);
	let uploadProgress = $state(0);
	let uploadedBytes = $state(0);
	let totalBytes = $state(0);
	let uploadSpeed = $state(0); // bytes/sec
	let etaSeconds = $state<number | null>(null);
	let error = $state('');
	let success = $state(false);
	let fileId = $state('');
	let statusMessage = $state('');

	function formatBytes(bytes: number): string {
		if (bytes <= 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatSpeed(bytesPerSec: number): string {
		if (bytesPerSec <= 0) return '0 KB/s';
		return `${formatBytes(bytesPerSec)}/s`;
	}

	function formatEta(seconds: number | null): string {
		if (seconds === null || !isFinite(seconds) || seconds <= 0) return '--';
		if (seconds < 60) return `${Math.ceil(seconds)}s left`;
		const mins = Math.floor(seconds / 60);
		const secs = Math.ceil(seconds % 60);
		return `${mins}m ${secs}s left`;
	}

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
		uploadedBytes = 0;
		totalBytes = file.size;
		uploadSpeed = 0;
		etaSeconds = null;
		statusMessage = 'Requesting direct upload URL...';

		try {
			// 1. Get the direct upload URL from the Aura API
			const initRes = await fetchApi('/videos/get-upload-url');
			const targetUploadUrl = initRes?.url || initRes?.result?.url || initRes?.uploadUrl;
			if (!targetUploadUrl) {
				throw new Error('Failed to obtain direct upload URL from service.');
			}

			// 2. Directly upload the video file to the upload URL with real-time progress & speed
			statusMessage = 'Uploading video...';
			const uploadResult = await new Promise<any>((resolve, reject) => {
				const xhr = new XMLHttpRequest();
				xhr.open('POST', targetUploadUrl);

				let lastLoaded = 0;
				let lastTime = performance.now();
				const speedSamples: number[] = [];

				xhr.upload.onloadstart = (event) => {
					lastTime = performance.now();
					lastLoaded = 0;
					uploadedBytes = 0;
					totalBytes = event.total || file!.size;
				};

				xhr.upload.onprogress = (event) => {
					if (event.lengthComputable) {
						const now = performance.now();
						uploadedBytes = event.loaded;
						totalBytes = event.total;
						uploadProgress = Math.round((event.loaded / event.total) * 100);

						const timeDiff = (now - lastTime) / 1000;
						if (timeDiff >= 0.2) {
							const bytesDiff = event.loaded - lastLoaded;
							const currentSpeed = bytesDiff / timeDiff;

							speedSamples.push(currentSpeed);
							if (speedSamples.length > 5) speedSamples.shift();
							const avgSpeed = speedSamples.reduce((a, b) => a + b, 0) / speedSamples.length;

							uploadSpeed = avgSpeed;
							const remainingBytes = Math.max(0, event.total - event.loaded);
							etaSeconds = avgSpeed > 0 ? remainingBytes / avgSpeed : null;

							lastLoaded = event.loaded;
							lastTime = now;
						}
					}
				};

				xhr.onload = () => {
					if (xhr.status >= 200 && xhr.status < 300) {
						uploadProgress = 100;
						uploadedBytes = totalBytes;
						uploadSpeed = 0;
						etaSeconds = 0;
						try {
							const json = JSON.parse(xhr.responseText);
							resolve(json);
						} catch {
							resolve(xhr.responseText);
						}
					} else {
						reject(new Error(`Upload service returned HTTP ${xhr.status}: ${xhr.statusText || 'Error'}`));
					}
				};

				xhr.onerror = () => reject(new Error('Network error occurred while uploading video.'));
				xhr.ontimeout = () => reject(new Error('Upload timed out. Please try again.'));

				const formData = new FormData();
				formData.append('file', file!);
				xhr.send(formData);
			});

			// Extract file ID returned by the upload service
			const finalVideoId =
				uploadResult?.result?.id ||
				uploadResult?.id ||
				uploadResult?.videoId ||
				(typeof uploadResult === 'string' && uploadResult.trim());

			if (!finalVideoId) {
				throw new Error('Upload completed, but no video ID was received from the service.');
			}

			// 3. Create the record in Aurahub database
			statusMessage = 'Finalizing video publishing...';
			const finalData = new FormData();
			finalData.append('videoId', finalVideoId);
			finalData.append('title', title);
			finalData.append('description', description);
			finalData.append('category', category);
			finalData.append('visibility', 'public');
			finalData.append('isShort', isShort.toString());
			finalData.append('isAdult', isAdult.toString());

			await fetchApi('/videos/create-record', {
				method: 'POST',
				body: finalData
			});

			fileId = finalVideoId;
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
						finalData.append('isAdult', isAdult.toString());

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
							uploadedBytes = loaded;
							totalBytes = total;
							uploadProgress = Math.floor((loaded / total) * 100);
							statusMessage = `Remote transfer in progress...`;
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
		isAdult = false;
		videoUrl = '';
		isUploading = false;
		uploadProgress = 0;
		uploadedBytes = 0;
		totalBytes = 0;
		uploadSpeed = 0;
		etaSeconds = null;
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
							class="bg-destructive/10 text-destructive flex items-center justify-between rounded-lg p-4 text-sm font-medium"
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
								<Label for="url"
									>Remote URL (Direct MP4 / Stream URL) <span class="text-destructive">*</span
									></Label
								>
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
										<div
											class="flex flex-col items-center justify-center px-4 pt-5 pb-6 text-center"
										>
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

						<div class="flex flex-col gap-3">
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
							<div class="flex items-center space-x-2">
								<input
									type="checkbox"
									id="isAdult"
									bind:checked={isAdult}
									disabled={isUploading}
									class="border-input h-4 w-4 rounded text-primary focus:ring-primary disabled:opacity-50"
								/>
								<Label for="isAdult" class="cursor-pointer">Contains Adult Content</Label>
							</div>
						</div>
					</div>

					{#if isUploading}
						<div class="rounded-2xl border border-primary/25 bg-muted/40 p-5 shadow-sm space-y-4">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
										<UploadCloud class="h-5 w-5 animate-bounce" />
									</div>
									<div>
										<p class="text-sm font-bold text-foreground">{statusMessage}</p>
										<p class="text-xs text-muted-foreground truncate max-w-[220px] sm:max-w-md">
											{file?.name || videoUrl}
										</p>
									</div>
								</div>
								<div class="text-right">
									<span class="font-mono text-2xl font-black text-primary">{uploadProgress}%</span>
								</div>
							</div>

							<!-- Smooth animated gradient bar -->
							<div class="relative h-3 w-full overflow-hidden rounded-full bg-muted">
								<div
									class="relative h-full rounded-full bg-gradient-to-r from-primary/80 via-primary to-primary transition-all duration-200 ease-out"
									style="width: {uploadProgress}%"
								>
									<div class="absolute inset-0 bg-white/20 animate-pulse"></div>
								</div>
							</div>

							<!-- Metrics Grid: Uploaded of Total, Speed, and Estimated Time -->
							<div class="grid grid-cols-3 gap-2 border-t border-border/60 pt-3 text-xs">
								<div class="space-y-0.5">
									<span class="text-muted-foreground block text-[11px] font-medium">Uploaded</span>
									<span class="font-mono font-bold text-foreground">
										{formatBytes(uploadedBytes)} / {formatBytes(totalBytes || file?.size || 0)}
									</span>
								</div>
								<div class="space-y-0.5 text-center">
									<span class="text-muted-foreground block text-[11px] font-medium">Upload Speed</span>
									<span class="font-mono font-bold text-foreground">
										{formatSpeed(uploadSpeed)}
									</span>
								</div>
								<div class="space-y-0.5 text-right">
									<span class="text-muted-foreground block text-[11px] font-medium">Time Remaining</span>
									<span class="font-mono font-bold text-foreground">
										{formatEta(etaSeconds)}
									</span>
								</div>
							</div>
						</div>
					{/if}

					<div class="pt-4">
						<Button
							type="submit"
							disabled={isUploading || (uploadType === 'direct' && !file)}
							class="text-md h-12 w-full"
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
