import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { Router, RouterModule } from '@angular/router'; // RouterModule for routerLink
import { AuthService } from '../../services/auth'; // Corrected path

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule], // Ensure RouterModule is imported
  templateUrl: './register.html',
  styleUrls: ['./register.css']
})
export class RegisterComponent {
  userData = {
    username: '',
    email: '',
    password: '',
    confirmPassword: ''
  };
  errorMessage: string = '';
  successMessage: string = '';

  constructor(private authService: AuthService, private router: Router) {}

  register(): void {
    this.errorMessage = '';
    this.successMessage = '';

    if (this.userData.password !== this.userData.confirmPassword) {
      this.errorMessage = 'Passwords do not match.';
      return;
    }

    // Basic email validation (Angular's built-in email validator handles more)
    if (!this.userData.email) {
      this.errorMessage = 'Email is required.';
      return;
    }

    // Basic password length (Angular's minlength validator handles this in template)
    if (this.userData.password.length < 6) {
      this.errorMessage = 'Password must be at least 6 characters long.';
      return;
    }

    const { confirmPassword, ...registrationData } = this.userData; // Exclude confirmPassword

    this.authService.register(registrationData).subscribe({
      next: (response) => {
        console.log('Registration successful', response);
        this.successMessage = 'Registration successful! Please login.';
        // Optionally redirect to login after a short delay or user action
        setTimeout(() => {
          this.router.navigate(['/login']);
        }, 2000); // Redirect after 2 seconds
      },
      error: (error: Error) => {
        this.errorMessage = error.message;
        console.error('Registration failed:', error);
      }
    });
  }
}
