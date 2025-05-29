import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { of, throwError } from 'rxjs';
import { By } from '@angular/platform-browser';

import { RegisterComponent } from './register';
import { AuthService } from '../../services/auth';
import { CommonModule } from '@angular/common';

describe('RegisterComponent', () => {
  let component: RegisterComponent;
  let fixture: ComponentFixture<RegisterComponent>;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let router: Router;

  beforeEach(async () => {
    mockAuthService = jasmine.createSpyObj('AuthService', ['register']);

    await TestBed.configureTestingModule({
      imports: [
        CommonModule,
        FormsModule,
        RouterTestingModule.withRoutes([
           // { path: 'login', component: class DummyLoginComponent {} } // For navigation test
        ]),
        RegisterComponent // Import standalone component
      ],
      providers: [
        { provide: AuthService, useValue: mockAuthService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(RegisterComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    fixture.detectChanges(); // Initial binding
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with empty user data', () => {
    expect(component.userData.username).toBe('');
    expect(component.userData.email).toBe('');
    expect(component.userData.password).toBe('');
    expect(component.userData.confirmPassword).toBe('');
  });

  it('should have a disabled register button when form is pristine or invalid', () => {
    const registerButton = fixture.debugElement.query(By.css('button[type="submit"]')).nativeElement;
    // Initially, required fields are empty, making the form invalid.
    expect(registerButton.disabled).toBeTrue(); 
  });

  it('should enable register button when required fields are filled and passwords match', fakeAsync(() => {
    const emailInput = fixture.debugElement.query(By.css('input[name="email"]')).nativeElement;
    const passwordInput = fixture.debugElement.query(By.css('input[name="password"]')).nativeElement;
    const confirmPasswordInput = fixture.debugElement.query(By.css('input[name="confirmPassword"]')).nativeElement;
    const registerButton = fixture.debugElement.query(By.css('button[type="submit"]')).nativeElement;

    emailInput.value = 'test@example.com';
    emailInput.dispatchEvent(new Event('input'));
    passwordInput.value = 'password123';
    passwordInput.dispatchEvent(new Event('input'));
    confirmPasswordInput.value = 'password123'; // Passwords match
    confirmPasswordInput.dispatchEvent(new Event('input'));
    
    tick();
    fixture.detectChanges();

    expect(registerButton.disabled).toBeFalse();
  }));

  it('should keep register button disabled if passwords do not match', fakeAsync(() => {
    const emailInput = fixture.debugElement.query(By.css('input[name="email"]')).nativeElement;
    const passwordInput = fixture.debugElement.query(By.css('input[name="password"]')).nativeElement;
    const confirmPasswordInput = fixture.debugElement.query(By.css('input[name="confirmPassword"]')).nativeElement;
    const registerButton = fixture.debugElement.query(By.css('button[type="submit"]')).nativeElement;

    emailInput.value = 'test@example.com';
    emailInput.dispatchEvent(new Event('input'));
    passwordInput.value = 'password123';
    passwordInput.dispatchEvent(new Event('input'));
    confirmPasswordInput.value = 'password1234'; // Passwords do not match
    confirmPasswordInput.dispatchEvent(new Event('input'));
    
    tick();
    fixture.detectChanges();
    
    // The button's disabled state is [disabled]="registerForm.invalid || (userData.password !== userData.confirmPassword && confirmPasswordInput.dirty)"
    // So if confirmPasswordInput is dirty and passwords don't match, it will be disabled.
    expect(registerButton.disabled).toBeTrue();
  }));
  
  it('should call AuthService.register on form submission with valid data (excluding confirmPassword)', fakeAsync(() => {
    const mockResponse = { id: '1', email: 'test@example.com', username: 'testuser' };
    mockAuthService.register.and.returnValue(of(mockResponse));
    spyOn(router, 'navigate'); // Spy on router navigation

    component.userData = {
      username: 'testuser',
      email: 'test@example.com',
      password: 'password123',
      confirmPassword: 'password123'
    };
    
    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    const form = fixture.debugElement.query(By.css('form'));
    form.triggerEventHandler('ngSubmit', null);
    tick(); // For async operations within register method

    const expectedPayload = {
      username: 'testuser',
      email: 'test@example.com',
      password: 'password123'
      // confirmPassword should not be part of this
    };
    expect(mockAuthService.register).toHaveBeenCalledWith(expectedPayload);
  }));

  it('should display success message and navigate to login on successful registration', fakeAsync(() => {
    const mockResponse = { id: '1', email: 'test@example.com', username: 'testuser' };
    mockAuthService.register.and.returnValue(of(mockResponse));
    spyOn(router, 'navigate');

    component.userData = {
      username: 'testuser',
      email: 'test@example.com',
      password: 'password123',
      confirmPassword: 'password123'
    };
    fixture.detectChanges(); tick(); fixture.detectChanges();

    component.register(); // Call method directly for easier control in test
    tick(); // for service call
    fixture.detectChanges(); // for success message

    expect(component.successMessage).toBe('Registration successful! Please login.');
    const successDiv = fixture.debugElement.query(By.css('.success-message'));
    expect(successDiv).toBeTruthy();
    expect(successDiv.nativeElement.textContent.trim()).toBe('Registration successful! Please login.');

    tick(2000); // For the setTimeout before navigation
    expect(router.navigate).toHaveBeenCalledWith(['/login']);
  }));
  
  it('should display error message if registration fails', fakeAsync(() => {
    const errorMessage = 'Email already exists';
    mockAuthService.register.and.returnValue(throwError(() => new Error(errorMessage)));

    component.userData = {
      username: 'testuser',
      email: 'test@example.com',
      password: 'password123',
      confirmPassword: 'password123'
    };
    fixture.detectChanges(); tick(); fixture.detectChanges();

    component.register();
    tick();
    fixture.detectChanges();

    expect(component.errorMessage).toBe(errorMessage);
    const errorDiv = fixture.debugElement.query(By.css('.error-message.api-error'));
    expect(errorDiv).toBeTruthy();
    expect(errorDiv.nativeElement.textContent.trim()).toBe(errorMessage);
  }));

  it('should display "Passwords do not match." if passwords mismatch on submit', () => {
    component.userData = {
      username: 'testuser',
      email: 'test@example.com',
      password: 'password123',
      confirmPassword: 'password456' // Mismatch
    };
    fixture.detectChanges();

    component.register();
    fixture.detectChanges();

    expect(component.errorMessage).toBe('Passwords do not match.');
    expect(mockAuthService.register).not.toHaveBeenCalled();
  });

  it('should require password to be at least 6 characters on submit (component logic check)', () => {
    component.userData = {
      username: 'testuser',
      email: 'test@example.com',
      password: '123', // Too short
      confirmPassword: '123'
    };
    fixture.detectChanges();
    component.register();
    fixture.detectChanges();
    expect(component.errorMessage).toBe('Password must be at least 6 characters long.');
    expect(mockAuthService.register).not.toHaveBeenCalled();
  });

});
