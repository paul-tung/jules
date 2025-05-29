import { Component } from '@angular/core';
import { CommonModule } from '@angular/common'; // For *ngIf, async pipe
import { RouterModule } from '@angular/router'; // For routerLink
import { Observable } from 'rxjs';
import { AuthService } from '../../services/auth'; // Corrected path

@Component({
  selector: 'app-navigation',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './navigation.html',
  styleUrls: ['./navigation.css']
})
export class NavigationComponent {
  isAuthenticated$: Observable<boolean>;
  // userEmail: string | null = null; // Placeholder for user email if you want to display it

  constructor(private authService: AuthService) {
    this.isAuthenticated$ = this.authService.isAuthenticated$;
    // Example of how you might get user info if stored in AuthService or decoded from token
    // this.authService.currentUser.subscribe(user => this.userEmail = user ? user.email : null);
  }

  logout(): void {
    this.authService.logout();
  }
}
