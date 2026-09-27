<script>
  import { onMount } from 'svelte';
  import { isTokenValid } from '$lib/utils/auth';
  
  let email = '';
  let password = '';
  let error = null;
  let loading = false;
  let showPassword = false;
  let emailFocused = false;
  let passwordFocused = false;
  let darkMode = false;
  let themeChanged = 0;
  
  const ADMIN_EMAIL = 'admin@email.com';
  const ADMIN_PASSWORD = 'pass';
  
  onMount(() => {
    if (isTokenValid()) {
      window.location.href = '/';
    }
    
    if (typeof window !== 'undefined') {
      darkMode = localStorage.getItem('intecs-theme') === 'dark';
    }
  });
  
  $: {
    if (themeChanged) {
      applyTheme();
    }
  }
  
  function applyTheme() {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', darkMode ? 'dark' : 'light');
    }
    localStorage.setItem('intecs-theme', darkMode ? 'dark' : 'light');
  }
  
  function toggleTheme() {
    darkMode = !darkMode;
    themeChanged += 1;
  }
  
  async function handleLogin(e) {
    e.preventDefault();
    error = null;
    loading = true;
    
    try {
      await new Promise(resolve => setTimeout(resolve, 600));
      
      if (email === ADMIN_EMAIL && password === ADMIN_PASSWORD) {
        const token = btoa(JSON.stringify({
          email: ADMIN_EMAIL,
          role: 'admin',
          exp: Date.now() + (24 * 60 * 60 * 1000)
        }));
        
        localStorage.setItem('token', token);
        localStorage.setItem('user', JSON.stringify({ email: ADMIN_EMAIL, role: 'admin' }));
        window.location.href = '/';
      } else {
        error = 'Invalid email or password';
      }
    } catch (e) {
      error = 'Connection failed';
    } finally {
      loading = false;
    }
  }
  
  function toggleShowPassword() {
    showPassword = !showPassword;
  }
</script>

<svelte:head>
  <title>Login - INTECS Monitoring</title>
</svelte:head>

<div class="login-page" data-theme={darkMode ? 'dark' : 'light'}>
  <div class="bg-pattern"></div>
  
  <button class="theme-toggle" on:click={toggleTheme} title="Toggle theme">
    {#if darkMode}
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="5"></circle>
        <line x1="12" y1="1" x2="12" y2="3"></line>
        <line x1="12" y1="21" x2="12" y2="23"></line>
        <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
        <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
        <line x1="1" y1="12" x2="3" y2="12"></line>
        <line x1="21" y1="12" x2="23" y2="12"></line>
        <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
        <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
      </svg>
    {:else}
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
      </svg>
    {/if}
  </button>
  
  <div class="login-container">
    <div class="login-header">
      <div class="logo-icon">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12l2 0M21 12l2 0M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"></path>
        </svg>
      </div>
      <h1 class="login-title">INTECS Monitoring</h1>
      <p class="login-subtitle">Fuel Monitoring System</p>
    </div>
    
    <form class="login-form" on:submit={handleLogin}>
      {#if error}
        <div class="login-error" in:fadeIn out:fadeOut>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
          <span>{error}</span>
        </div>
      {/if}
      
      <div class="form-group">
        <label for="email">Email Address</label>
        <div class="input-wrapper">
          <span class="input-icon" class:email-focus={emailFocused}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="2" y="4" width="20" height="16" rx="2"></rect>
              <path d="m22 7-8.97 5.7a1.9 1.9 0 0 1-2.06 0L2 7"></path>
            </svg>
          </span>
          <input 
            id="email"
            type="email" 
            placeholder="Enter your email"
            value={email}
            on:input={(e) => email = e.target.value}
            on:focus={() => emailFocused = true}
            on:blur={() => emailFocused = false}
            required
            autocomplete="email"
          />
        </div>
      </div>
      
      <div class="form-group">
        <label for="password">Password</label>
        <div class="input-wrapper">
          <span class="input-icon" class:password-focus={passwordFocused}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
            </svg>
          </span>
          <input 
            id="password"
            type="{showPassword ? 'text' : 'password'}" 
            placeholder="Enter your password"
            value={password}
            on:input={(e) => password = e.target.value}
            on:focus={() => passwordFocused = true}
            on:blur={() => passwordFocused = false}
            required
            autocomplete="current-password"
          />
          <button type="button" class="toggle-password-btn" on:click={toggleShowPassword} tabindex="-1">
            {#if showPassword}
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path>
                <line x1="1" y1="1" x2="23" y2="23"></line>
              </svg>
            {:else}
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                <circle cx="12" cy="12" r="3"></circle>
              </svg>
            {/if}
          </button>
        </div>
      </div>
      
      <button type="submit" class="login-btn" disabled={loading}>
        {#if loading}
          <span class="spinner"></span>
          Signing in...
        {:else}
          Sign In
        {/if}
      </button>
    </form>
    
    <div class="login-footer">
      <p class="credentials-hint">Default: admin@email.com / pass</p>
    </div>
  </div>
</div>

<style>
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
  }
  
  @keyframes fadeOut {
    from { opacity: 1; transform: translateY(0); }
    to { opacity: 0; transform: translateY(-8px); }
  }
  
  @keyframes slideUp {
    from { opacity: 0; transform: translateY(20px); }
    to { opacity: 1; transform: translateY(0); }
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
    padding: 1rem;
    font-family: inherit;
    position: relative;
    overflow: hidden;
  }
  
  .bg-pattern {
    position: absolute;
    inset: 0;
    background-image: 
      radial-gradient(circle at 20% 50%, rgba(59, 130, 246, 0.08) 0%, transparent 50%),
      radial-gradient(circle at 80% 80%, rgba(139, 92, 246, 0.08) 0%, transparent 50%);
    pointer-events: none;
  }
  
  .theme-toggle {
    position: absolute;
    top: 1.5rem;
    right: 1.5rem;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.15);
    color: #94a3b8;
    padding: 0.625rem;
    border-radius: 50%;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    backdrop-filter: blur(8px);
    z-index: 10;
  }
  
  .theme-toggle:hover {
    color: white;
    background: rgba(255, 255, 255, 0.15);
    border-color: rgba(255, 255, 255, 0.25);
    transform: scale(1.05);
  }
  
  .login-container {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(20px);
    border-radius: 20px;
    box-shadow: 0 25px 80px rgba(0, 0, 0, 0.35), 0 0 0 1px rgba(255, 255, 255, 0.1);
    width: 100%;
    max-width: 420px;
    overflow: hidden;
    animation: slideUp 0.5s ease-out;
    border: 1px solid rgba(255, 255, 255, 0.15);
  }
  
  [data-theme='dark'] .login-container {
    background: rgba(30, 41, 59, 0.95);
    box-shadow: 0 25px 80px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255, 255, 255, 0.05);
  }
  
  .login-header {
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    color: white;
    padding: 2.75rem 2rem 2rem;
    text-align: center;
    position: relative;
    overflow: hidden;
  }
  
  .login-header::before {
    content: '';
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, transparent 60%);
    pointer-events: none;
  }
  
  .logo-icon {
    font-size: 3rem;
    margin-bottom: 0.75rem;
    filter: brightness(100%) saturate(0%);
    animation: pulse 2s ease-in-out infinite;
  }
  
  @keyframes pulse {
    0%, 100% { transform: scale(1); }
    50% { transform: scale(1.05); }
  }
  
  .login-title {
    font-size: 1.75rem;
    font-weight: 700;
    margin: 0 0 0.25rem 0;
    letter-spacing: -0.02em;
    position: relative;
  }
  
  .login-subtitle {
    font-size: 0.95rem;
    opacity: 0.9;
    margin: 0;
    font-weight: 400;
    position: relative;
  }
  
  .login-form {
    padding: 2rem;
  }
  
  .form-group {
    margin-bottom: 1.25rem;
  }
  
  .form-group label {
    display: block;
    font-size: 0.875rem;
    font-weight: 600;
    color: #334155;
    margin-bottom: 0.5rem;
    letter-spacing: 0.01em;
  }
  
  [data-theme='dark'] .form-group label {
    color: #cbd5e1;
  }
  
  .input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }
  
  .input-icon {
    position: absolute;
    left: 0.875rem;
    top: 50%;
    transform: translateY(-50%);
    color: #94a3b8;
    pointer-events: none;
    transition: color 0.2s ease;
  }
  
  .input-wrapper input {
    width: 100%;
    padding: 0.875rem 3rem 0.875rem 2.75rem;
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    font-size: 0.95rem;
    color: #1e293b;
    background: #f8fafc;
    transition: all 0.2s ease;
    font-family: inherit;
  }
  
  [data-theme='dark'] .input-wrapper input {
    color: #f1f5f9;
    background: #0f172a;
    border-color: #334155;
  }
  
  .input-wrapper input:focus {
    outline: none;
    border-color: #3b82f6;
    background: white;
    box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.12);
  }
  
  [data-theme='dark'] .input-wrapper input:focus {
    background: #1e293b;
    border-color: #3b82f6;
    box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.2);
  }
  
  .input-icon.email-focus {
    color: #3b82f6;
  }
  
  .input-icon.password-focus {
    color: #3b82f6;
  }
  
  .toggle-password-btn {
    position: absolute;
    right: 0.625rem;
    background: none;
    border: none;
    cursor: pointer;
    color: #94a3b8;
    padding: 0.375rem;
    transition: all 0.2s ease;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .toggle-password-btn:hover {
    color: #64748b;
    background: rgba(0, 0, 0, 0.05);
  }
  
  [data-theme='dark'] .toggle-password-btn:hover {
    color: #cbd5e1;
    background: rgba(255, 255, 255, 0.05);
  }
  
  .login-error {
    background: #fef2f2;
    border: 1px solid #fecaca;
    color: #dc2626;
    padding: 0.875rem 1rem;
    border-radius: 10px;
    font-size: 0.875rem;
    margin-bottom: 1.25rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    animation: fadeIn 0.3s ease;
  }
  
  [data-theme='dark'] .login-error {
    background: rgba(220, 38, 38, 0.1);
    border-color: rgba(220, 38, 38, 0.3);
  }
  
  .login-btn {
    width: 100%;
    padding: 0.875rem;
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    color: white;
    border: none;
    border-radius: 12px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    font-family: inherit;
    margin-top: 0.5rem;
    position: relative;
    overflow: hidden;
  }
  
  .login-btn::before {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, transparent 100%);
    opacity: 0;
    transition: opacity 0.2s ease;
  }
  
  .login-btn:hover:not(:disabled)::before {
    opacity: 1;
  }
  
  .login-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 8px 25px rgba(59, 130, 246, 0.4);
  }
  
  .login-btn:active:not(:disabled) {
    transform: translateY(0);
  }
  
  .login-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
  
  .spinner {
    display: inline-block;
    width: 18px;
    height: 18px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  
  .login-footer {
    padding: 1.5rem 2rem 2rem;
    text-align: center;
    border-top: 1px solid #e2e8f0;
  }
  
  [data-theme='dark'] .login-footer {
    border-top-color: #334155;
  }
  
  .credentials-hint {
    font-size: 0.8rem;
    color: #94a3b8;
    margin: 0;
    padding: 0.75rem;
    background: #f8fafc;
    border-radius: 8px;
    border: 1px dashed #cbd5e1;
    transition: all 0.2s ease;
  }
  
  [data-theme='dark'] .credentials-hint {
    background: #0f172a;
    border-color: #334155;
    color: #64748b;
  }
  
  @media (max-width: 480px) {
    .login-container {
      border-radius: 16px;
    }
    
    .login-header {
      padding: 2rem 1.5rem 1.5rem;
    }
    
    .login-form {
      padding: 1.5rem;
    }
    
    .login-footer {
      padding: 1.25rem 1.5rem 1.5rem;
    }
    
    .theme-toggle {
      top: 1rem;
      right: 1rem;
    }
  }
</style>
