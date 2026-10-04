<script lang="ts">
    import { Mail, Lock, User, ArrowRight } from 'lucide-svelte';

    let username = $state('');
    let email = $state('');
    let password = $state('');
    let loading = $state(false);

    async function handleRegister(e: Event) {
        e.preventDefault();
        loading = true;
        // Mock request to our Go backend
        setTimeout(() => {
            loading = false;
        }, 1000);
    }
</script>

<div class="min-h-[80vh] flex items-center justify-center p-4">
    <div class="w-full max-w-md glass-card p-8 flex flex-col gap-6 relative overflow-hidden shadow-2xl shadow-black/5 dark:shadow-white/5">
        <div class="absolute -top-24 -left-24 w-48 h-48 bg-primary opacity-5 rounded-full blur-3xl pointer-events-none"></div>

        <div class="flex flex-col gap-2 relative z-10 text-center">
            <h1 class="text-3xl font-bold text-text-main tracking-tight">Create Account</h1>
            <p class="text-text-muted text-sm">Join Aurahub to upload and interact with videos</p>
        </div>

        <form onsubmit={handleRegister} class="flex flex-col gap-4 relative z-10 mt-2">
            <div class="space-y-1.5">
                <label for="username" class="text-sm font-medium text-text-main">Username</label>
                <div class="relative group">
                    <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <User class="h-5 w-5 text-text-muted group-focus-within:text-primary transition-colors" />
                    </div>
                    <input 
                        id="username" 
                        type="text" 
                        bind:value={username}
                        required
                        class="block w-full pl-10 pr-4 py-2.5 border border-border rounded-xl bg-surface/50 text-text-main placeholder-text-muted focus:ring-1 focus:ring-primary focus:border-primary focus:bg-surface focus:outline-none transition-all duration-300"
                        placeholder="coolcreator99"
                    >
                </div>
            </div>

            <div class="space-y-1.5">
                <label for="email" class="text-sm font-medium text-text-main">Email</label>
                <div class="relative group">
                    <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <Mail class="h-5 w-5 text-text-muted group-focus-within:text-primary transition-colors" />
                    </div>
                    <input 
                        id="email" 
                        type="email" 
                        bind:value={email}
                        required
                        class="block w-full pl-10 pr-4 py-2.5 border border-border rounded-xl bg-surface/50 text-text-main placeholder-text-muted focus:ring-1 focus:ring-primary focus:border-primary focus:bg-surface focus:outline-none transition-all duration-300"
                        placeholder="you@example.com"
                    >
                </div>
            </div>

            <div class="space-y-1.5">
                <label for="password" class="text-sm font-medium text-text-main">Password</label>
                <div class="relative group">
                    <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <Lock class="h-5 w-5 text-text-muted group-focus-within:text-primary transition-colors" />
                    </div>
                    <input 
                        id="password" 
                        type="password" 
                        bind:value={password}
                        required
                        class="block w-full pl-10 pr-4 py-2.5 border border-border rounded-xl bg-surface/50 text-text-main placeholder-text-muted focus:ring-1 focus:ring-primary focus:border-primary focus:bg-surface focus:outline-none transition-all duration-300"
                        placeholder="••••••••"
                    >
                </div>
            </div>

            <button 
                type="submit" 
                disabled={loading}
                class="mt-4 w-full flex items-center justify-center gap-2 bg-primary text-background hover:bg-primary-hover px-4 py-3 rounded-xl font-medium transition-all duration-300 disabled:opacity-70 disabled:cursor-not-allowed group shadow-md"
            >
                {#if loading}
                    <div class="w-5 h-5 border-2 border-background border-t-transparent rounded-full animate-spin"></div>
                    <span>Creating account...</span>
                {:else}
                    <span>Sign Up</span>
                    <ArrowRight class="w-4 h-4 group-hover:translate-x-1 transition-transform" />
                {/if}
            </button>
        </form>

        <p class="text-center text-sm text-text-muted mt-2">
            Already have an account? <a href="/login" class="text-text-main font-medium hover:underline transition-all">Sign in</a>
        </p>
    </div>
</div>
