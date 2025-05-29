import { Component } from '@angular/core';
import { RouterModule } from '@angular/router'; // For <router-outlet>
import { NavigationComponent } from './components/navigation/navigation'; // Corrected path
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-root', // Standard selector for the root component
  standalone: true,
  imports: [
    CommonModule, // Common directives like *ngIf, *ngFor
    RouterModule, // For <router-outlet> and routerLink directives
    NavigationComponent // The navigation component
  ],
  templateUrl: './app.html', // Path to your HTML file
  styleUrls: ['./app.css']   // Path to your CSS file
})
export class AppComponent {
  title = 'auth-frontend';
}
