import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common'; // For *ngIf
import { AuthService } from '../../services/auth'; // Corrected path
import { jwtDecode } from 'jwt-decode'; // Import jwt-decode

interface DecodedToken {
  email: string;
  user_id: string;
  exp: number;
  iat: number;
  iss: string;
}

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './dashboard.html',
  styleUrls: ['./dashboard.css']
})
export class DashboardComponent implements OnInit {
  userEmail: string | null = null;

  constructor(private authService: AuthService) {}

  ngOnInit(): void {
    const token = this.authService.getToken();
    if (token) {
      try {
        const decodedToken: DecodedToken = jwtDecode(token);
        this.userEmail = decodedToken.email;
      } catch (error) {
        console.error('Error decoding JWT:', error);
        // Handle error, e.g., logout user if token is invalid
        this.authService.logout();
      }
    }
  }
}
