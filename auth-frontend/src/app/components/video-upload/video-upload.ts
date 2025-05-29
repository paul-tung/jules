import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClientModule, HttpEventType, HttpResponse } from '@angular/common/http'; // HttpClientModule for service, HttpEventType for progress
import { VideoService } from '../../services/video'; // Corrected path

@Component({
  selector: 'app-video-upload',
  standalone: true,
  imports: [CommonModule, FormsModule, HttpClientModule], // HttpClientModule might be needed if not globally provided sufficiently
  templateUrl: './video-upload.html',
  styleUrls: ['./video-upload.css']
})
export class VideoUploadComponent {
  selectedFile: File | null = null;
  videoTitle: string = '';
  videoDescription: string = '';
  videoTags: string = ''; // Comma-separated tags

  uploadProgress: number = 0;
  successMessage: string = '';
  errorMessage: string = '';
  uploadInProgress: boolean = false;

  constructor(private videoService: VideoService) {}

  onFileSelected(event: Event): void {
    const element = event.currentTarget as HTMLInputElement;
    let fileList: FileList | null = element.files;
    if (fileList && fileList.length > 0) {
      this.selectedFile = fileList[0];
      this.successMessage = ''; // Clear previous messages
      this.errorMessage = '';
      this.uploadProgress = 0;
    } else {
      this.selectedFile = null;
    }
  }

  onUpload(): void {
    if (!this.selectedFile) {
      this.errorMessage = 'Please select a video file to upload.';
      return;
    }

    this.successMessage = '';
    this.errorMessage = '';
    this.uploadProgress = 0;
    this.uploadInProgress = true;

    const tagsArray = this.videoTags.split(',').map(tag => tag.trim()).filter(tag => tag !== '');

    this.videoService.uploadVideo(this.selectedFile, this.videoTitle, this.videoDescription, tagsArray)
      .subscribe({
        next: (event) => {
          if (event.type === HttpEventType.UploadProgress) {
            if (event.total) {
              this.uploadProgress = Math.round(100 * (event.loaded / event.total));
            }
          } else if (event instanceof HttpResponse) { // Check for HttpResponse
            this.successMessage = 'Video uploaded successfully!';
            console.log('Upload successful', event.body);
            this.resetForm();
            this.uploadInProgress = false;
          }
        },
        error: (err: Error) => {
          this.errorMessage = `Upload failed: ${err.message}`;
          console.error('Upload error:', err);
          this.uploadProgress = 0;
          this.uploadInProgress = false;
        }
      });
  }

  private resetForm(): void {
    this.selectedFile = null;
    this.videoTitle = '';
    this.videoDescription = '';
    this.videoTags = '';
    // Reset file input visually if possible (tricky, often involves re-rendering or specific tricks)
    // For simplicity, the user will have to re-select if they want to upload another immediately.
    const fileInput = document.getElementById('videoFile') as HTMLInputElement;
    if (fileInput) {
      fileInput.value = ''; // This attempts to clear it
    }
  }
}
