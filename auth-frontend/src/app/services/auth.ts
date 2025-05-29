import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Router } from '@angular/router';
import { BehaviorSubject, Observable, throwError } from 'rxjs';
import { catchError, tap } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface AuthResponse {
  token?: string;
  // Add other expected fields from backend response, e.g., user details
  email?: string;
  username?: string;
  ID?: string;
  error?: string; // For error messages from backend
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private apiUrl = environment.apiUrl + '/auth';
  private readonly JWT_TOKEN = 'JWT_TOKEN';

  private isAuthenticatedSubject = new BehaviorSubject<boolean>(this.hasToken());
  public isAuthenticated$: Observable<boolean> = this.isAuthenticatedSubject.asObservable();

  constructor(private http: HttpClient, private router: Router) {}

  private hasToken(): boolean {
    return !!this.getToken();
  }

  register(user: any): Observable<any> {
    return this.http.post<AuthResponse>(`${this.apiUrl}/register`, user).pipe(
      catchError(this.handleError)
    );
  }

  login(credentials: any): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.apiUrl}/login`, credentials).pipe(
      tap(response => {
        if (response && response.token) {
          this.storeToken(response.token);
          this.isAuthenticatedSubject.next(true);
          this.router.navigate(['/dashboard']);
        }
      }),
      catchError(this.handleError)
    );
  }

  logout(): void {
    this.removeToken();
    this.isAuthenticatedSubject.next(false);
    this.router.navigate(['/login']);
  }

  getToken(): string | null {
    return localStorage.getItem(this.JWT_TOKEN);
  }

  isAuthenticatedUser(): boolean {
    return this.isAuthenticatedSubject.value;
  }

  private storeToken(token: string): void {
    localStorage.setItem(this.JWT_TOKEN, token);
  }

  private removeToken(): void {
    localStorage.removeItem(this.JWT_TOKEN);
  }

  private handleError(error: HttpErrorResponse) {
    let errorMessage = 'An unknown error occurred!';
    if (error.error instanceof ErrorEvent) {
      // Client-side errors
      errorMessage = `Error: ${error.error.message}`;
    } else {
      // Server-side errors
      if (error.status === 0) {
        errorMessage = 'Could not connect to the server. Please try again later.';
      } else if (error.error && error.error.error) {
        errorMessage = error.error.error; // Use specific error from backend
      } else {
        errorMessage = `Server returned code: ${error.status}, error message is: ${error.message}`;
      }
    }
    return throwError(() => new Error(errorMessage));
  }
}
