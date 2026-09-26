<script>
  import { onMount } from 'svelte';
  
  let email = '';
  let password = '';
  let error = null;
  let loading = false;
  let showPassword = false;
  
  const ADMIN_EMAIL = 'admin@email.com';
  const ADMIN_PASSWORD = 'pass';
  
  onMount(() => {
    if (localStorage.getItem('token')) {
      window.location.href = '/';
    }
  });
  
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

<div class="login-page">
  <div class="login-container">
    <div class="login-header">
      <div class="login-logo">&#9881;</div>
      <h1 class="login-title">INTECS Monitoring</h1>
      <p class="login-subtitle">Fuel Monitoring System</p>
    </div>
    
    <form class="login-form" on:submit={handleLogin}>
      {#if error}
        <div class="login-error">
          <span class="error-icon">&#9888;&#65039;</span>
          <span>{error}</span>
        </div>
      {/if}
      
      <div class="form-group">
        <label for="email">Email Address</label>
        <div class="input-wrapper">
          <span class="input-icon">&#9993;</span>
          <input 
            id="email"
            type="email" 
            placeholder="Enter your email"
            value={email}
            on:input={(e) => email = e.target.value}
            required
            autocomplete="email"
          />
        </div>
      </div>
      
      <div class="form-group">
        <label for="password">Password</label>
        <div class="input-wrapper">
          <span class="input-icon">&#128274;</span>
          <input 
            id="password"
            type="{showPassword ? 'text' : 'password'}" 
            placeholder="Enter your password"
            value={password}
            on:input={(e) => password = e.target.value}
            required
            autocomplete="current-password"
          />
          <button type="button" class="toggle-password-btn" on:click={toggleShowPassword}>
            {showPassword ? '&#128065;' : '&#128064;'}
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
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #334155 100%);
    padding: 1rem;
    font-family: inherit;
  }
  
  .login-container {
    background: rgba(255, 255, 255, 0.95);
    border-radius: 16px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    width: 100%;
    max-width: 420px;
    overflow: hidden;
  }
  
  .login-header {
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    color: white;
    padding: 2.5rem 2rem 2rem;
    text-align: center;
  }
  
  .login-logo {
    font-size: 3rem;
    margin-bottom: 0.75rem;
    filter: brightness(100%) saturate(0%);
  }
  
  .login-title {
    font-size: 1.75rem;
    font-weight: 700;
    margin: 0 0 0.25rem 0;
    letter-spacing: -0.02em;
  }
  
  .login-subtitle {
    font-size: 0.95rem;
    opacity: 0.9;
    margin: 0;
    font-weight: 400;
  }
  
  .login-form {
    padding: 2rem;
  }
  
  .form-group {
    margin-bottom: 1.25rem;
  }
  
  .form-group label {
    display: block;
    font-size: 0.9rem;
    font-weight: 600;
    color: #334155;
    margin-bottom: 0.5rem;
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
    font-size: 1.1rem;
    pointer-events: none;
  }
  
  .input-wrapper input {
    width: 100%;
    padding: 0.875rem 0.875rem 0.875rem 2.75rem;
    border: 2px solid #e2e8f0;
    border-radius: 10px;
    font-size: 0.95rem;
    color: #1e293b;
    background: #f8fafc;
    transition: all 0.2s ease;
    font-family: inherit;
  }
  
  .input-wrapper input:focus {
    outline: none;
    border-color: #3b82f6;
    background: white;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }
  
  .toggle-password-btn {
    position: absolute;
    right: 0.75rem;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 1.1rem;
    color: #94a3b8;
    padding: 0.25rem;
    transition: color 0.2s ease;
  }
  
  .toggle-password-btn:hover {
    color: #64748b;
  }
  
  .login-error {
    background: #fef2f2;
    border: 1px solid #fecaca;
    color: #dc2626;
    padding: 0.875rem 1rem;
    border-radius: 10px;
    font-size: 0.9rem;
    margin-bottom: 1.25rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  
  .error-icon {
    font-style: normal;
  }
  
  .login-btn {
    width: 100%;
    padding: 0.875rem;
    background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
    color: white;
    border: none;
    border-radius: 10px;
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
  }
  
  .login-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 8px 20px rgba(59, 130, 246, 0.4);
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
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .login-footer {
    padding: 1.5rem 2rem 2rem;
    text-align: center;
    border-top: 1px solid #e2e8f0;
  }
  
  .credentials-hint {
    font-size: 0.8rem;
    color: #94a3b8;
    margin: 0;
    padding: 0.75rem;
    background: #f8fafc;
    border-radius: 8px;
    border: 1px dashed #cbd5e1;
  }
  
  @media (max-width: 480px) {
    .login-container {
      border-radius: 12px;
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
  }
</style>
