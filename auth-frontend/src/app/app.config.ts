import { ApplicationConfig, importProvidersFrom } from '@angular/core';
import { provideRouter } from '@angular/router';
import { HttpClientModule, HTTP_INTERCEPTORS, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'; // Added HttpClientModule for older Angular versions if needed, and withInterceptorsFromDi

import { routes } from './app.routes';
import { TokenInterceptor } from './interceptors/token-interceptor'; // Ensure this path is correct
// No need to import AuthService or AuthGuard here if they are providedIn: 'root'

export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    importProvidersFrom(HttpClientModule), // Necessary for older Angular versions, good practice for some DI scenarios
    provideHttpClient(withInterceptorsFromDi()), // Modern way to provide HttpClient with interceptor support
    {
      provide: HTTP_INTERCEPTORS,
      useClass: TokenInterceptor,
      multi: true
    }
    // AuthService and AuthGuard are typically provided in root automatically.
    // If not, they would be added here:
    // AuthService,
    // AuthGuard
  ]
};
