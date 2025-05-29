import { Injectable } from '@angular/core';
import {
  ActivatedRouteSnapshot,
  RouterStateSnapshot,
  Router,
  UrlTree
} from '@angular/router';
import { Observable } from 'rxjs';
import { AuthService } from '../services/auth'; // Corrected path

@Injectable({
  providedIn: 'root'
})
export class AuthGuard  { // Removed CanActivate as it's deprecated for functional guards

  constructor(private authService: AuthService, private router: Router) {}

  canActivate(
    next: ActivatedRouteSnapshot,
    state: RouterStateSnapshot): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
    
    if (this.authService.isAuthenticatedUser()) {
      return true;
    } else {
      // Redirect to the login page
      return this.router.createUrlTree(['/login']);
    }
  }
}
