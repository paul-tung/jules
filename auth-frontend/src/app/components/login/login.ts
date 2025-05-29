import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { Router, RouterModule } from '@angular/router'; // RouterModule for routerLink
import { AuthService, AuthResponse } from '../../services/auth'; // Corrected path

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule], // Ensure RouterModule is imported
  templateUrl: './login.html',
  styleUrls: ['./login.css']
})
export class LoginComponent {
  credentials = {
    email: '',
    password: ''
  };
  errorMessage: string = '';

  constructor(private authService: AuthService, private router: Router) {}

  login(): void {
    this.errorMessage = ''; // Clear previous error messages
    if (!this.credentials.email || !this.credentials.password) {
      this.errorMessage = 'Email and password are required.';
      return;
    }

    this.authService.login(this.credentials).subscribe({
      next: (response: AuthResponse) => {
        // AuthService's tap operator handles token storage and navigation
        // If login is successful, navigation to dashboard is handled by the service
        console.log('Login successful', response);
      },
      error: (error: Error) => { // Explicitly type error as Error
        this.errorMessage = error.message; // Display error message from AuthService
        console.error('Login failed:', error);
      }
    });
  }
}
